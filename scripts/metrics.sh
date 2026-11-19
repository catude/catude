#!/bin/sh
# Archives the numbers that matter for growth, once a day (GitHub only keeps 14 days of traffic).
#   scripts/metrics.sh OWNER/REPO >> metrics.csv      (needs: gh auth login, with push access to the repo)
# Columns: date, stars, forks, open_issues, views_14d, unique_visitors_14d, clones_14d, unique_cloners_14d,
#          release_downloads, top_referrer
set -eu
repo="${1:?usage: metrics.sh OWNER/REPO}"
d=$(date -u +%F)
r=$(gh api "repos/$repo" --jq '[.stargazers_count,.forks_count,.open_issues_count]|@csv')
v=$(gh api "repos/$repo/traffic/views" --jq '[.count,.uniques]|@csv')
c=$(gh api "repos/$repo/traffic/clones" --jq '[.count,.uniques]|@csv')
dl=$(gh api "repos/$repo/releases" --jq '[.[].assets[].download_count]|add // 0')
ref=$(gh api "repos/$repo/traffic/popular/referrers" --jq '.[0].referrer // ""')
echo "$d,$r,$v,$c,$dl,$ref"
