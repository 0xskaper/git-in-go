set -e

tmpFile=$(mktemp)

( cd $(dirname "$0") &&
	go build -o "$tmpFile"  )

exec "$tmpFile" "$@"
