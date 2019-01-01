$ErrorActionPreference = 'Stop'
$StartTime=Get-Date

Try {
    Write-Host -ForegroundColor Cyan "`nINFO: executeCI.ps1 starting at $(date)`n"
    Write-Host  -ForegroundColor Green "INFO: Script version $SCRIPT_VER"

    # Turn off progress bars
	$global:ProgressPreference='SilentlyContinue'


    # Make sure TESTRUN_DRIVE is set
    if ($env:TESTRUN_DRIVE -eq $Null) { Throw "ERROR: Environment variable TESTRUN_DRIVE is not set" }

    # Make sure TESTRUN_SUBDIR is set
    if ($env:TESTRUN_SUBDIR -eq $Null) { Throw "ERROR: Environment variable TESTRUN_SUBDIR is not set" }

    $env:CITEMP="$env:TESTRUN_DRIVE`:\$env:TESTRUN_SUBDIR\CI-$env:GIT_COMMIT"
    New-Item -ItemType Directory "$env:CITEMP" -ErrorAction SilentlyContinue | Out-Null
    New-Item -ItemType Directory "$env:CITEMP\binary" -ErrorAction SilentlyContinue | Out-Null

    # Build the image
    Write-Host  -ForegroundColor Cyan "`n`nINFO: Building the image from Dockerfile.windows at $(Get-Date)..."
    Write-Host
    $ErrorActionPreference = "SilentlyContinue"
    $Duration=$(Measure-Command { docker build -t docker -f Dockerfile.windows . | Out-Host })
    $ErrorActionPreference = "Stop"
    if (-not($LastExitCode -eq 0)) {
        Throw "ERROR: Failed to build image from Dockerfile.windows"
    }
    Write-Host  -ForegroundColor Green "INFO: Image build ended at $(Get-Date). Duration`:$Duration"

    Write-Host  -ForegroundColor Cyan "`n`nINFO: Creating and starting container $(Get-Date)..."
    $ContainerID = docker run --name $env:GIT_COMMIT -v "$env:CITEMP\binary:c:\binary" -e GIT_COMMIT=$env:GIT_COMMIT -e PR=$env:PR --rm --detach=true docker powershell -Command 'Write-Host Container ready!;Start-Sleep -Seconds 14400'
    Write-Host  -ForegroundColor Green "INFO: Creating and starting container ended at $(Get-Date). Duration`:$Duration"

    Write-Host  -ForegroundColor Cyan "`n`nINFO: Running git on container at $(Get-Date)..."
    $Duration=$(Measure-Command { docker exec $ContainerID powershell -Command '& c:\git.ps1' | Out-Host })
    $ErrorActionPreference = "Stop"
    if (-not($LastExitCode -eq 0)) {
        Throw "ERROR: Failed to run git"
    }
    Write-Host  -ForegroundColor Green "INFO: Running git ended at $(Get-Date). Duration`:$Duration"

    Write-Host  -ForegroundColor Cyan "`n`nINFO: Running build and tests on container at $(Get-Date)..."
    $Duration=$(Measure-Command { docker exec $ContainerID powershell -Command '& hack/make.ps1 -All' | Out-Host })
    $ErrorActionPreference = "Stop"
    if (-not($LastExitCode -eq 0)) {
        Throw "ERROR: Failed to run build/tests"
    }
    Write-Host  -ForegroundColor Green "INFO: Running tests ended at $(Get-Date). Duration`:$Duration"

    $ErrorActionPreference = "Stop"

    Write-Host -ForegroundColor Green "INFO: executeCI.ps1 Completed successfully at $(Get-Date)."
}
Catch [Exception] {
    $FinallyColour="Red"
    Write-Host -ForegroundColor Red ("`r`n`r`nERROR: Failed '$_' at $(Get-Date)")
    Write-Host -ForegroundColor Red ($_.InvocationInfo.PositionMessage)
    Write-Host "`n`n"

    # Exit to ensure Jenkins captures it. Don't do this in the ISE or interactive Powershell - they will catch the Throw onwards.
    if ( ([bool]([Environment]::GetCommandLineArgs() -Like '*-NonInteractive*')) -and `
         ([bool]([Environment]::GetCommandLineArgs() -NotLike "*Powershell_ISE.exe*"))) {
        exit 1
    }
    Throw $_
}
Finally {
    $ErrorActionPreference="SilentlyContinue"
	$global:ProgressPreference=$origProgressPreference
    Write-Host  -ForegroundColor Green "INFO: Tidying up at end of run"

    $Dur=New-TimeSpan -Start $StartTime -End $(Get-Date)
    Write-Host -ForegroundColor $FinallyColour "`nINFO: executeCI.ps1 exiting at $(date). Duration $dur`n"
}
