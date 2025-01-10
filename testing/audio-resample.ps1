# Define the directory and the program you want to call
$directoryPath = "C:\Users\LALT\Documents\Shows\8x10_2025\test"  # Change this to the directory path

$programPath = "ffmpeg"

# Get all files in the directory
$files = Get-ChildItem -Path $directoryPath -File

# Redirect output from subprocesses to current directory
$outputFile = ".\output.txt"
$errorFile = ".\error.txt"

# Loop through each file and call the program on it
foreach ($file in $files) {
    Write-Host "Processing $($file.FullName)"

    $name = $file.FullName
    $newName = $name + "_orig"
    $resampleName = $name + "_rs"

    Copy-Item -Path $file.FullName -Destination $newName
    
    # Call the program with the file as a parameter
    Start-Process $programPath -ArgumentList "-i", $newName, "-ar", "48000", "-f", "mp3", $resampleName -RedirectStandardOutput $outputFile -RedirectStandardError $errorFile -Wait

    Remove-Item $name
    Move-Item -Path $resampleName -Destination $name
}
