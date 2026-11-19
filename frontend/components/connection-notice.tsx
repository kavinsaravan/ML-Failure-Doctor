export default function ConnectionNotice({ error, lastUpdated, retrying, onRetry }: {
  error: string | null;
  lastUpdated: Date | null;
  retrying: boolean;
  onRetry: () => void;
}) {
  if (!error) return lastUpdated ? (
    <p className="text-sm text-slate-400 mb-4">Last updated: {lastUpdated.toLocaleTimeString()}</p>
  ) : null;
  return (
    <div role="alert" className="mb-6 rounded-lg border border-amber-600 bg-amber-950/40 p-4 text-amber-100">
      <p>{error}</p>
      <p className="mt-1 text-sm">
        {lastUpdated ? `Showing previously loaded data from ${lastUpdated.toLocaleTimeString()}.` : 'No data has been loaded. Check your connection and try again.'}
      </p>
      <button onClick={onRetry} disabled={retrying} className="mt-3 rounded bg-amber-700 px-4 py-2 text-white disabled:opacity-50">
        {retrying ? 'Retrying…' : 'Retry now'}
      </button>
    </div>
  );
}
