param(
    [Parameter(Mandatory = $true)][string] $ImagePath,
    [Parameter(Mandatory = $true)][string] $OutputPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Runtime.WindowsRuntime

[Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType = WindowsRuntime] > $null
[Windows.Media.Ocr.OcrResult, Windows.Foundation, ContentType = WindowsRuntime] > $null
[Windows.Globalization.Language, Windows.Foundation, ContentType = WindowsRuntime] > $null
[Windows.Storage.StorageFile, Windows.Storage, ContentType = WindowsRuntime] > $null
[Windows.Storage.Streams.IRandomAccessStream, Windows.Storage.Streams, ContentType = WindowsRuntime] > $null
[Windows.Graphics.Imaging.BitmapDecoder, Windows.Graphics.Imaging, ContentType = WindowsRuntime] > $null
[Windows.Graphics.Imaging.SoftwareBitmap, Windows.Graphics.Imaging, ContentType = WindowsRuntime] > $null

$AsTaskGenericMethod = [System.WindowsRuntimeSystemExtensions].GetMethods() |
    Where-Object {
        $_.Name -eq "AsTask" -and
        $_.IsGenericMethod -and
        $_.GetGenericArguments().Count -eq 1 -and
        $_.GetParameters().Count -eq 1
    } |
    Select-Object -First 1

if (-not $AsTaskGenericMethod) {
    throw "Windows Runtime task support could not be initialized."
}

function Await-WinRtResult {
    param(
        [Parameter(Mandatory = $true)][object] $Operation,
        [Parameter(Mandatory = $true)][Type] $ResultType
    )

    $Method = $AsTaskGenericMethod.MakeGenericMethod($ResultType)
    $Task = $Method.Invoke($null, @($Operation))
    $Task.Wait()

    return $Task.Result
}


function Test-IsRtlLanguage {
    param(
        [string] $LanguageTag
    )

    if ([string]::IsNullOrWhiteSpace($LanguageTag)) {
        return $false
    }

    $Normalized = $LanguageTag.ToLowerInvariant()

    return (
        $Normalized.StartsWith("ar") -or
        $Normalized.StartsWith("fa") -or
        $Normalized.StartsWith("ur") -or
        $Normalized.StartsWith("he")
    )
}

function Get-ArabicScriptCount {
    param(
        [string] $Text
    )

    if ([string]::IsNullOrWhiteSpace($Text)) {
        return 0
    }

    $Count = 0

    foreach ($Character in $Text.ToCharArray()) {
        $Code = [int][char]$Character

        if (
            ($Code -ge 0x0600 -and $Code -le 0x06FF) -or
            ($Code -ge 0x0750 -and $Code -le 0x077F) -or
            ($Code -ge 0x08A0 -and $Code -le 0x08FF) -or
            ($Code -ge 0xFB50 -and $Code -le 0xFDFF) -or
            ($Code -ge 0xFE70 -and $Code -le 0xFEFF)
        ) {
            $Count++
        }
    }

    return $Count
}

function Get-OrderedOcrText {
    param(
        [Parameter(Mandatory = $true)]
        [object] $Result,

        [Parameter(Mandatory = $true)]
        [string] $LanguageTag
    )

    $Lines = @($Result.Lines)

    if ($Lines.Count -eq 0) {
        return ([string]$Result.Text).Trim()
    }

    $IsRtl = Test-IsRtlLanguage -LanguageTag $LanguageTag
    $OutputLines = New-Object System.Collections.Generic.List[string]

    foreach ($Line in $Lines) {
        $Words = @($Line.Words)

        if ($Words.Count -eq 0) {
            continue
        }

        if ($IsRtl) {
            $Words = @(
                $Words |
                    Sort-Object `
                        @{ Expression = { [double]$_.BoundingRect.X }; Descending = $true },
                        @{ Expression = { [double]$_.BoundingRect.Y }; Descending = $false }
            )
        }
        else {
            $Words = @(
                $Words |
                    Sort-Object `
                        @{ Expression = { [double]$_.BoundingRect.X }; Descending = $false },
                        @{ Expression = { [double]$_.BoundingRect.Y }; Descending = $false }
            )
        }

        $LineText = (
            $Words |
                ForEach-Object { ([string]$_.Text).Trim() } |
                Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
        ) -join " "

        if (-not [string]::IsNullOrWhiteSpace($LineText)) {
            $OutputLines.Add($LineText.Trim())
        }
    }

    if ($OutputLines.Count -eq 0) {
        return ([string]$Result.Text).Trim()
    }

    return ($OutputLines -join "`r`n").Trim()
}

function Get-TextScore {
    param(
        [string] $Text
    )

    if ([string]::IsNullOrWhiteSpace($Text)) {
        return -1000000.0
    }

    $Score = 0.0

    foreach ($Character in $Text.ToCharArray()) {
        if ([char]::IsLetter($Character)) {
            $Score += 2.2
        }
        elseif ([char]::IsDigit($Character)) {
            $Score += 1.4
        }
        elseif ([char]::IsWhiteSpace($Character)) {
            $Score += 0.05
        }
        elseif ([char]::IsPunctuation($Character)) {
            $Score += 0.2
        }
        else {
            $Score -= 1.5
        }
    }

    return $Score
}

$AvailableLanguages = @([Windows.Media.Ocr.OcrEngine]::AvailableRecognizerLanguages)

if ($AvailableLanguages.Count -eq 0) {
    throw "No Windows OCR language is installed. Add English, French, Arabic, or another OCR-capable language in Settings > Time & language > Language & region."
}

$Stream = $null
$SoftwareBitmap = $null

try {
    $StorageFile = Await-WinRtResult `
        -Operation ([Windows.Storage.StorageFile]::GetFileFromPathAsync($ImagePath)) `
        -ResultType ([Windows.Storage.StorageFile])

    $Stream = Await-WinRtResult `
        -Operation ($StorageFile.OpenAsync([Windows.Storage.FileAccessMode]::Read)) `
        -ResultType ([Windows.Storage.Streams.IRandomAccessStream])

    $Decoder = Await-WinRtResult `
        -Operation ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($Stream)) `
        -ResultType ([Windows.Graphics.Imaging.BitmapDecoder])

    $SoftwareBitmap = Await-WinRtResult `
        -Operation ($Decoder.GetSoftwareBitmapAsync()) `
        -ResultType ([Windows.Graphics.Imaging.SoftwareBitmap])

    $BestText = ""
    $BestLanguage = ""
    $BestScore = -1000000.0

    foreach ($Language in $AvailableLanguages) {
        try {
            $Engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromLanguage($Language)

            if (-not $Engine) {
                continue
            }

            $Result = Await-WinRtResult `
                -Operation ($Engine.RecognizeAsync($SoftwareBitmap)) `
                -ResultType ([Windows.Media.Ocr.OcrResult])

            $LanguageTag = [string]$Language.LanguageTag
            $Text = Get-OrderedOcrText -Result $Result -LanguageTag $LanguageTag
            $Score = Get-TextScore -Text $Text

            $ArabicScriptCount = Get-ArabicScriptCount -Text $Text

            if (Test-IsRtlLanguage -LanguageTag $LanguageTag) {
                $Score += ($ArabicScriptCount * 0.8)
            }
            elseif ($ArabicScriptCount -gt 0) {
                $Score -= ($ArabicScriptCount * 0.35)
            }

            if ($Score -gt $BestScore) {
                $BestText = $Text
                $BestLanguage = $LanguageTag
                $BestScore = $Score
            }
        }
        catch {
            # Continue trying the remaining installed Windows OCR languages.
        }
    }

    if ([string]::IsNullOrWhiteSpace($BestLanguage)) {
        $BestLanguage = "automatic"
    }

    $Payload = "__MINIEXTRACTOR_LANGUAGE__=$BestLanguage`r`n$BestText"

    Set-Content `
        -LiteralPath $OutputPath `
        -Value $Payload `
        -Encoding UTF8
}
finally {
    if ($SoftwareBitmap -and $SoftwareBitmap -is [System.IDisposable]) {
        $SoftwareBitmap.Dispose()
    }

    if ($Stream -and $Stream -is [System.IDisposable]) {
        $Stream.Dispose()
    }
}
