$ErrorActionPreference = 'Stop'

$configOverrides = @('SERVER_HOST', 'SERVER_PORT', 'DATA_DIR')
$savedValues = @{}

foreach ($name in $configOverrides) {
    $savedValues[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
    [Environment]::SetEnvironmentVariable($name, $null, 'Process')
}

Push-Location $PSScriptRoot
try {
    & go run .\cmd\server
    exit $LASTEXITCODE
}
finally {
    Pop-Location
    foreach ($name in $configOverrides) {
        [Environment]::SetEnvironmentVariable($name, $savedValues[$name], 'Process')
    }
}
