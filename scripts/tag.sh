
version=$(go run ./cmd/redis-enforce-expire -version | awk '{ print $2 }' | awk -F= '{ print $2 }')

git tag v${version}
git tag chart-${version}
