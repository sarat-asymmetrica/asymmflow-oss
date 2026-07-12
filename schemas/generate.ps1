param(
    [switch]$CheckOnly,
    [switch]$TypeScript,
    [switch]$TypeScriptOnly
)

$ErrorActionPreference = "Stop"

$script:SchemaDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$script:RepoRoot = Split-Path -Parent $script:SchemaDir
$script:GoOutDir = Join-Path $script:SchemaDir "go"
$script:TsOutDir = Join-Path $script:RepoRoot "frontend\src\lib\types\schemas"
$script:SchemaOrder = @(
    "common",
    "finance",
    "crm",
    "butler",
    "documents",
    "infra",
    "sync"
)

function Resolve-CommandPath {
    param(
        [Parameter(Mandatory=$true)][string]$Name,
        [string[]]$Fallbacks = @()
    )

    $cmd = Get-Command $Name -ErrorAction SilentlyContinue
    if ($cmd) {
        return $cmd.Source
    }

    foreach ($fallback in $Fallbacks) {
        if (Test-Path -LiteralPath $fallback) {
            return $fallback
        }
    }

    throw "Required command not found: $Name"
}

function Resolve-GoCapnpStd {
    if ($env:CAPNP_GO_STD -and (Test-Path -LiteralPath $env:CAPNP_GO_STD)) {
        return (Resolve-Path -LiteralPath $env:CAPNP_GO_STD).Path
    }

    $candidates = @()
    $goPath = (& go env GOPATH 2>$null)
    if ($LASTEXITCODE -eq 0 -and $goPath) {
        $modRoot = Join-Path $goPath "pkg\mod"
        $candidates += Get-ChildItem -LiteralPath $modRoot -Directory -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -eq "capnproto.org" } |
            ForEach-Object { Get-ChildItem -LiteralPath (Join-Path $_.FullName "go\capnp") -Directory -ErrorAction SilentlyContinue } |
            ForEach-Object { Get-ChildItem -LiteralPath $_.FullName -Directory -ErrorAction SilentlyContinue } |
            Where-Object { $_.Name -like "v3@*" } |
            ForEach-Object { Join-Path $_.FullName "std" } |
            Where-Object { Test-Path -LiteralPath (Join-Path $_ "go.capnp") }

        $candidates += Get-ChildItem -LiteralPath (Join-Path $modRoot "capnproto.org\go") -Directory -ErrorAction SilentlyContinue |
            Where-Object { $_.Name -like "capnp@*" } |
            ForEach-Object { Join-Path $_.FullName "std" } |
            Where-Object { Test-Path -LiteralPath (Join-Path $_ "go.capnp") }
    }

    $known = "C:\Users\YourName\go\pkg\mod\capnproto.org\go\capnp\v3@v3.1.0-alpha.2\std"
    if (Test-Path -LiteralPath (Join-Path $known "go.capnp")) {
        $candidates += $known
    }

    $std = $candidates | Sort-Object -Descending | Select-Object -First 1
    if (-not $std) {
        throw "Could not find go.capnp. Set CAPNP_GO_STD to the directory containing go.capnp."
    }
    return (Resolve-Path -LiteralPath $std).Path
}

function Assert-ChildPath {
    param(
        [Parameter(Mandatory=$true)][string]$Child,
        [Parameter(Mandatory=$true)][string]$Parent
    )

    $resolvedParent = (Resolve-Path -LiteralPath $Parent).Path.TrimEnd('\')
    if (Test-Path -LiteralPath $Child) {
        $resolvedChild = (Resolve-Path -LiteralPath $Child).Path
    } else {
        $resolvedChild = [System.IO.Path]::GetFullPath($Child)
    }
    if (-not $resolvedChild.StartsWith($resolvedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Refusing to write outside expected directory: $resolvedChild"
    }
}

function Invoke-CapnpGo {
    $capnp = Resolve-CommandPath "capnp" @("C:\ProgramData\chocolatey\bin\capnp.exe")
    $capnpcGo = Resolve-CommandPath "capnpc-go" @("C:\Users\YourName\go\bin\capnpc-go.exe")
    $goStd = Resolve-GoCapnpStd
    $tmpRoot = if ($env:GOTMPDIR) { $env:GOTMPDIR } elseif ($env:TEMP) { $env:TEMP } else { [System.IO.Path]::GetTempPath() }
    $tmpOut = Join-Path $tmpRoot ("asymmflow-capnp-" + [System.Guid]::NewGuid().ToString("N"))

    New-Item -ItemType Directory -Force -Path $tmpOut | Out-Null
    try {
        $schemaPaths = $script:SchemaOrder | ForEach-Object { Join-Path $script:SchemaDir "$_.capnp" }
        foreach ($schemaPath in $schemaPaths) {
            if (-not (Test-Path -LiteralPath $schemaPath)) {
                throw "Missing schema: $schemaPath"
            }
        }

        Write-Host "Using capnp: $capnp"
        Write-Host "Using capnpc-go: $capnpcGo"
        Write-Host "Using go.capnp import path: $goStd"
        & $capnp compile "--import-path=$goStd" "--import-path=$script:SchemaDir" "-ogo:$tmpOut" @schemaPaths
        if ($LASTEXITCODE -ne 0) {
            throw "capnp compile failed"
        }

        if ($CheckOnly) {
            Write-Host "Cap'n Proto schemas compile successfully."
            return
        }

        Assert-ChildPath $script:GoOutDir $script:SchemaDir
        if (Test-Path -LiteralPath $script:GoOutDir) {
            Remove-Item -LiteralPath $script:GoOutDir -Recurse -Force
        }
        New-Item -ItemType Directory -Force -Path $script:GoOutDir | Out-Null

        Get-ChildItem -LiteralPath $tmpOut -Recurse -Filter "*.go" | ForEach-Object {
            $content = Get-Content -LiteralPath $_.FullName -Raw
            if ($content -notmatch '(?m)^package\s+([A-Za-z_][A-Za-z0-9_]*)\s*$') {
                throw "Could not determine generated Go package for $($_.FullName)"
            }
            $pkg = $Matches[1]
            $pkgDir = Join-Path $script:GoOutDir $pkg
            New-Item -ItemType Directory -Force -Path $pkgDir | Out-Null
            Move-Item -LiteralPath $_.FullName -Destination (Join-Path $pkgDir $_.Name) -Force
        }

        Write-Host "Generated Go schemas in $script:GoOutDir"
    } finally {
        if (Test-Path -LiteralPath $tmpOut) {
            Remove-Item -LiteralPath $tmpOut -Recurse -Force
        }
    }
}

function Convert-TypeScriptType {
    param([Parameter(Mandatory=$true)][string]$TypeName)

    $t = $TypeName.Trim()
    if ($t -match '^List\((.+)\)$') {
        return "$(Convert-TypeScriptType $Matches[1])[]"
    }

    switch ($t) {
        "Text" { return "string" }
        "Bool" { return "boolean" }
        "Data" { return "Uint8Array" }
        "Float64" { return "number" }
        "Int64" { return "number" }
        "Int32" { return "number" }
        "UInt64" { return "number" }
        "UInt32" { return "number" }
        default { return ($t -replace '^Common\.', 'Common.' -replace '^CRM\.', 'Crm.' -replace '^Finance\.', 'Finance.' -replace '^Butler\.', 'Butler.' -replace '^Documents\.', 'Documents.' -replace '^Infra\.', 'Infra.' -replace '^Sync\.', 'Sync.') }
    }
}

function Convert-NamespaceName {
    param([Parameter(Mandatory=$true)][string]$SchemaName)
    switch ($SchemaName) {
        "crm" { return "Crm" }
        "sync" { return "Sync" }
        default {
            return (Get-Culture).TextInfo.ToTitleCase($SchemaName)
        }
    }
}

function Convert-CapnpToTypeScript {
    param(
        [Parameter(Mandatory=$true)][string]$SchemaName,
        [Parameter(Mandatory=$true)][string]$Source
    )

    $namespace = Convert-NamespaceName $SchemaName
    $lines = New-Object System.Collections.Generic.List[string]
    $lines.Add("export namespace $namespace {")

    $currentKind = $null
    $currentName = $null
    $enumValues = @()

    foreach ($rawLine in ($Source -split "`r?`n")) {
        $line = ($rawLine -replace '#.*$', '').Trim()
        if ($line -eq "") { continue }

        if (-not $currentKind -and $line -match '^enum\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{') {
            $currentKind = "enum"
            $currentName = $Matches[1]
            $enumValues = @()
            continue
        }

        if (-not $currentKind -and $line -match '^struct\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{') {
            $currentKind = "struct"
            $currentName = $Matches[1]
            $lines.Add("")
            $lines.Add("  export interface $currentName {")
            continue
        }

        if ($currentKind -and $line -match '^\}') {
            if ($currentKind -eq "enum") {
                $union = if ($enumValues.Count -gt 0) { ($enumValues | ForEach-Object { "'$_'" }) -join " | " } else { "never" }
                $lines.Add("")
                $lines.Add("  export type $currentName = $union;")
            } elseif ($currentKind -eq "struct") {
                $lines.Add("  }")
            }
            $currentKind = $null
            $currentName = $null
            $enumValues = @()
            continue
        }

        if ($currentKind -eq "enum" -and $line -match '^([A-Za-z_][A-Za-z0-9_]*)\s+@[0-9]+;') {
            $enumValues += $Matches[1]
            continue
        }

        if ($currentKind -eq "struct" -and $line -match '^([A-Za-z_][A-Za-z0-9_]*)\s+@[0-9]+\s+:(.+);') {
            $fieldName = $Matches[1]
            $fieldType = Convert-TypeScriptType $Matches[2]
            $lines.Add("    $fieldName`: $fieldType;")
        }
    }

    $lines.Add("}")
    return ($lines -join "`r`n")
}

function Invoke-TypeScriptGeneration {
    Assert-ChildPath $script:TsOutDir (Join-Path $script:RepoRoot "frontend")
    if (Test-Path -LiteralPath $script:TsOutDir) {
        Remove-Item -LiteralPath $script:TsOutDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $script:TsOutDir | Out-Null

    $out = New-Object System.Collections.Generic.List[string]
    $out.Add("// Generated from schemas/*.capnp - DO NOT EDIT.")
    $out.Add("// Generated by schemas/generate.ps1.")
    $out.Add("")

    foreach ($schema in $script:SchemaOrder) {
        $path = Join-Path $script:SchemaDir "$schema.capnp"
        $source = Get-Content -LiteralPath $path -Raw
        $out.Add((Convert-CapnpToTypeScript $schema $source))
        $out.Add("")
    }

    $indexPath = Join-Path $script:TsOutDir "index.ts"
    Set-Content -LiteralPath $indexPath -Value ($out -join "`r`n") -Encoding UTF8
    Write-Host "Generated TypeScript schemas in $script:TsOutDir"
}

if (-not $TypeScriptOnly) {
    Invoke-CapnpGo
}

if ($TypeScript -or $TypeScriptOnly) {
    Invoke-TypeScriptGeneration
}
