Docker v19.03.12-olljanat
================

This branch is created from https://github.com/docker/engine/tree/v19.03.12 and cherry-picked PRs are merged to it because of release process is too slow.

This branch will be dropped after needed features/fixes are released as part of official version.


# Usage
You can update Docker to latest custom build with commands:
```powershell
Install-Module -Name DockerMsftProvider -Repository PSGallery -Force
Register-PackageSource -ProviderName DockerMsftProvider -Name "olljanat-custom-builds" -Location "https://raw.githubusercontent.com/olljanat/moby/v19.03.12-olljanat/DockerOlljanatIndex.json"
Install-Package -Name docker -ProviderName DockerMsftProvider -Update -Force -RequiredVersion "19.03.12-olljanat1"
```
