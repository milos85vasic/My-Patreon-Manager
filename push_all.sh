#!/bin/bash
set -e

echo "=== Pushing to all upstreams ==="

# GitHub
echo "1. GitHub..."
if git push github main 2>&1 | grep -q "rejected"; then
    echo "   ⚠️  Push rejected, trying force..."
    git push github main --force
else
    echo "   ✅ Success"
fi

# GitVerse
echo "2. GitVerse..."
if git push gitverse main 2>&1 | grep -q "rejected"; then
    echo "   ⚠️  Push rejected, trying force..."
    git push gitverse main --force
else
    echo "   ✅ Success"
fi

# GitLab - check if we need to fetch first
echo "3. GitLab..."
git fetch gitlab
if git diff main gitlab/main --quiet; then
    echo "   ✅ Already up to date"
else
    echo "   📥 GitLab has new commits, pulling..."
    git pull gitlab main --no-rebase || echo "   ⚠️  Pull failed, may need manual merge"
    git push gitlab main 2>&1 | grep -q "rejected" && echo "   ⚠️  Push rejected (branch protected?)" || echo "   ✅ Success"
fi

# GitFlic
echo "4. GitFlic..."
if git push gitflic main 2>&1 | grep -q "rejected"; then
    echo "   ⚠️  Push rejected, trying force..."
    git push gitflic main --force 2>&1 | grep -q "rejected" && echo "   ❌ Force push also rejected" || echo "   ✅ Success"
else
    echo "   ✅ Success"
fi

echo "=== Done ==="