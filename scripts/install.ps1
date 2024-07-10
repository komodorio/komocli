function Get-OS {
    if ($IsLinux) {
        "linux"
    }
    elseif ($IsMacOS) {
        "darwin"
    }
    elseif ($IsWindows) {
        "windows"
    }
    else {
    "windows"
    }
}

function Get-Arch {
    $systemType = (Get-CimInstance -ClassName CIM_ComputerSystem).SystemType
    if ($systemType -match 'x64') {
        "amd64"
    }
    elseif ($systemType -match 'ARM') {
        "arm64"
    }
    elseif ($systemType -match 'x86') {
        "386"
    }
}

function Get-DownloadURL {
    $os = Get-OS
    $arch = Get-Arch
    $url = "https://api.github.com/repos/komodorio/komocli/releases/latest"
    $response = Invoke-RestMethod -Uri $url
    $browserDownloadURL = ($response.assets | Where-Object { $_.browser_download_url -like "*${os}_${arch}*" }).browser_download_url
    return $browserDownloadURL
}

function Get-Version {
    $url = "https://api.github.com/repos/komodorio/komocli/releases/latest"
    $response = Invoke-RestMethod -Uri $url
    return $response.name
}

function AddTo-Path{
    param(
        [string]$Dir
    )
    if (!(Test-Path $Dir) ){
        Write-warning "Supplied directory was not found!"
        return
    }
    $PATH = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    if ($PATH -notlike "*"+$Dir+"*" ){
        [Environment]::SetEnvironmentVariable("PATH", "$PATH;$Dir", "Machine")
    }
}

$os = Get-OS
$arch = Get-Arch
$downloadURL = Get-DownloadURL
$version = Get-Version

Write-Host $os
Write-Host $arch
Write-Host $downloadURL
Write-Host "Downloading komocli package..."
Invoke-WebRequest -Uri $downloadURL -OutFile "komocli.exe"
Write-Host "Installing komocli..."
mkdir $env:APPDATA\komodor
$installation_path = "$env:APPDATA\komodor"
Move-Item -Path "komocli.exe" -Destination $installation_path
AddTo-Path($installation_path)
Write-Host "komocli installation completed!"
