"""Issue, list, rotate, or revoke individual CrashLens keys as the operator."""
import argparse
import getpass
import json
import os
import urllib.request


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--api-url", default=os.getenv("CRASHLENS_URL", "http://localhost:8080"))
    commands = parser.add_subparsers(dest="command", required=True)
    issue = commands.add_parser("issue")
    issue.add_argument("name")
    issue.add_argument("--owner-id", help="Existing owner's ID when issuing a replacement key")
    commands.add_parser("list")
    revoke = commands.add_parser("revoke")
    revoke.add_argument("id")
    args = parser.parse_args()
    token = os.getenv("CRASHLENS_API_KEY") or getpass.getpass("Operator CrashLens API key: ")
    url = args.api_url.rstrip("/") + "/api-keys"
    body = None
    method = "GET"
    if args.command == "issue":
        method = "POST"
        body = json.dumps({"name": args.name, "owner_id": args.owner_id or ""}).encode()
    elif args.command == "revoke":
        method = "DELETE"
        url += "/" + args.id
    request = urllib.request.Request(url, data=body, method=method, headers={
        "Authorization": "Bearer " + token, "Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=15) as response:
        data = response.read()
    if args.command == "issue":
        print("Save this user key now; it cannot be retrieved later.")
    print(json.dumps(json.loads(data), indent=2) if data else "Key revoked.")


if __name__ == "__main__":
    main()
