find . \
  -type d \( -name target -o -name .git \) -prune -o \
  -type f \( -name "*.go" -o -name "*.conf" \) -print | while read -r f; do
    echo "===== $f ====="
    cat "$f"
    echo
done
