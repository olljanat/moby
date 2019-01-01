# Checkout and merge PR
<#
git init C:\gopath\src\github.com\docker\docker
git --version
git fetch --tags --progress https://github.com/moby/moby.git +refs/heads/*:refs/remotes/origin/*
git config remote.origin.url https://github.com/moby/moby.git
git config --add remote.origin.fetch +refs/heads/*:refs/remotes/origin/*
git config remote.origin.url https://github.com/moby/moby.git
git fetch --tags --progress https://github.com/moby/moby.git +refs/heads/*:refs/remotes/origin/*
git rev-parse "origin/master^{commit}"
git config core.sparsecheckout
git checkout -f $env:GIT_COMMIT
git rev-list --no-walk $env:GIT_COMMIT
git fetch origin "+refs/pull/$($env:PR)/head:refs/remotes/origin/pr/$($env:PR)"
git merge "origin/pr/$($env:PR)"
#>

git clone https://github.com/olljanat/moby.git . -b win-ci-on-docker