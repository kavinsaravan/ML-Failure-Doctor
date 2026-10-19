"""Local terminal telemetry awaiting delivery; never stores SDK credentials."""
import hashlib
import json
import os
from pathlib import Path
import tempfile


class PendingUploads:
    def __init__(self, api_url, api_key, directory=None):
        root = directory or os.getenv('CRASHLENS_OUTBOX_DIR')
        if root is None:
            root = Path(os.getenv('XDG_CACHE_HOME', str(Path.home() / '.cache'))) / 'crashlens' / 'outbox'
        # Bind replay to the same destination and key without writing either key.
        namespace = hashlib.sha256((api_url + '\0' + (api_key or '')).encode()).hexdigest()
        self.directory = Path(root) / namespace

    def save(self, record):
        self.directory.mkdir(parents=True, exist_ok=True, mode=0o700)
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', dir=self.directory,
                                             prefix='.upload-', delete=False) as file:
                temporary = Path(file.name)
                json.dump(record, file)
                file.flush()
                os.fsync(file.fileno())
            destination = self.directory / (record['client_run_id'] + '.json')
            temporary.replace(destination)
            return destination
        finally:
            if temporary is not None and temporary.exists():
                temporary.unlink()

    def paths(self):
        return sorted(self.directory.glob('*.json'))

    @staticmethod
    def load(path):
        record = json.loads(path.read_text())
        if record.get('schema_version') != 1 or record['payload'].get('status') not in ('failed', 'succeeded'):
            raise ValueError('Invalid pending terminal upload')
        return record

    @staticmethod
    def remove(path):
        try:
            path.unlink()
        except FileNotFoundError:
            pass
