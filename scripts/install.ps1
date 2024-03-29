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
    if ((Get-WmiObject -Class Win32_ComputerSystem).SystemType -match '(x64)') {
        "amd64"
    }
    elseif ((Get-WmiObject -Class Win32_ComputerSystem).SystemType -match 'arm') {
        "arm64"
    }
    elseif ((Get-WmiObject -Class Win32_ComputerSystem).SystemType -match '386') {
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

$os = Get-OS
$arch = Get-Arch
$downloadURL = Get-DownloadURL
$version = Get-Version

Write-Host $os
Write-Host $arch
Write-Host $downloadURL
Write-Host "Downloading komocli package..."
Invoke-WebRequest -Uri $downloadURL -OutFile "komocli_${version}_${os}_${arch}.tar.gz"

Write-Host "Extracting komocli package..."
tar -xf komocli_${version}_${os}_${arch}.tar.gz

Write-Host "Installing komocli..."
Move-Item -Path "komocli.exe" -Destination $env:APPDATA
Remove-Item -Path "komocli_${version}_${os}_${arch}.tar.gz"
Write-Host "komocli installation completed!"