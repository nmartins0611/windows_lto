# SAS LTO Device Monitor Script
# Checks for attached SAS LTO tape drives and sends email notifications

param(
    [string]$SMTPServer = "smtp.gmail.com",
    [int]$SMTPPort = 587,
    [string]$FromEmail = "your-email@gmail.com",
    [string]$ToEmail = "admin@company.com",
    [string]$EmailUsername = "your-email@gmail.com",
    [string]$EmailPassword = "your-app-password",
    [switch]$SendOnSuccess = $false,
    [switch]$SendOnFailure = $true
)

# Function to send email
function Send-EmailNotification {
    param(
        [string]$Subject,
        [string]$Body,
        [string]$Priority = "Normal"
    )
    
    try {
        $SecurePassword = ConvertTo-SecureString $EmailPassword -AsPlainText -Force
        $Credential = New-Object System.Management.Automation.PSCredential($EmailUsername, $SecurePassword)
        
        $MailParams = @{
            SmtpServer = $SMTPServer
            Port = $SMTPPort
            UseSsl = $true
            Credential = $Credential
            From = $FromEmail
            To = $ToEmail
            Subject = $Subject
            Body = $Body
            BodyAsHtml = $true
            Priority = $Priority
        }
        
        Send-MailMessage @MailParams
        Write-Host "Email sent successfully" -ForegroundColor Green
        return $true
    }
    catch {
        Write-Host "Failed to send email: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
}

# Function to check for LTO devices
function Test-LTODevices {
    $LTODevices = @()
    $DeviceFound = $false
    $ATTOCardFound = $false
    
    try {
        # Check for tape drives using WMI
        Write-Host "Checking for ATTO SAS card and LTO devices..." -ForegroundColor Yellow
        
        # Method 1: Check for ATTO SAS Controllers first
        Write-Host "  Scanning for ATTO SAS controllers..." -ForegroundColor Gray
        $ATTOControllers = Get-WmiObject -Class Win32_SCSIController -ErrorAction SilentlyContinue | 
            Where-Object { $_.Name -match "ATTO|ExpressSAS" -or $_.Description -match "ATTO|ExpressSAS" }
        
        if ($ATTOControllers) {
            $ATTOCardFound = $true
            Write-Host "  Found ATTO SAS controller(s)" -ForegroundColor Green
            foreach ($Controller in $ATTOControllers) {
                $DeviceInfo = [PSCustomObject]@{
                    DeviceID = $Controller.DeviceID
                    Name = $Controller.Name
                    Description = $Controller.Description
                    Status = $Controller.Status
                    Availability = $Controller.Availability
                    Manufacturer = $Controller.Manufacturer
                    MediaType = "ATTO SAS Controller"
                    Method = "ATTO SAS Controller"
                }
                $LTODevices += $DeviceInfo
            }
        } else {
            Write-Host "  No ATTO SAS controllers found" -ForegroundColor Yellow
        }
        
        # Method 2: Check Win32_TapeDrive for LTO devices
        Write-Host "  Scanning for tape drives..." -ForegroundColor Gray
        $TapeDrives = Get-WmiObject -Class Win32_TapeDrive -ErrorAction SilentlyContinue
        if ($TapeDrives) {
            foreach ($Drive in $TapeDrives) {
                $DeviceInfo = [PSCustomObject]@{
                    DeviceID = $Drive.DeviceID
                    Name = $Drive.Name
                    Description = $Drive.Description
                    Status = $Drive.Status
                    Availability = $Drive.Availability
                    Manufacturer = $Drive.Manufacturer
                    MediaType = $Drive.MediaType
                    Method = "Win32_TapeDrive"
                }
                $LTODevices += $DeviceInfo
                $DeviceFound = $true
            }
            Write-Host "  Found $($TapeDrives.Count) tape drive(s)" -ForegroundColor Green
        } else {
            Write-Host "  No tape drives found via Win32_TapeDrive" -ForegroundColor Yellow
        }
        
        # Method 3: Check SCSI devices for any additional LTO devices
        Write-Host "  Scanning SCSI devices for LTO..." -ForegroundColor Gray
        $SCSIDevices = Get-WmiObject -Class Win32_SCSIController -ErrorAction SilentlyContinue
        if ($SCSIDevices) {
            $LTOSCSIDevices = $SCSIDevices | Where-Object { 
                ($_.Name -match "LTO|Tape" -or $_.Description -match "LTO|Tape") -and
                ($_.Name -notmatch "ATTO|ExpressSAS") # Avoid duplicating ATTO controllers
            }
            
            foreach ($Controller in $LTOSCSIDevices) {
                $DeviceInfo = [PSCustomObject]@{
                    DeviceID = $Controller.DeviceID
                    Name = $Controller.Name
                    Description = $Controller.Description
                    Status = $Controller.Status
                    Availability = $Controller.Availability
                    Manufacturer = $Controller.Manufacturer
                    MediaType = "SCSI LTO Controller"
                    Method = "Win32_SCSIController"
                }
                $LTODevices += $DeviceInfo
                $DeviceFound = $true
            }
        }
        
        # Method 4: Check PnP devices for LTO/Tape devices
        Write-Host "  Scanning PnP devices for LTO..." -ForegroundColor Gray
        $PnPDevices = Get-WmiObject -Class Win32_PnPEntity -ErrorAction SilentlyContinue | Where-Object {
            ($_.Name -match "LTO|Tape|ATTO|ExpressSAS" -or $_.Description -match "LTO|Tape|ATTO|ExpressSAS") -and
            $_.Status -eq "OK"
        }
        
        foreach ($Device in $PnPDevices) {
            # Check if this device is already in our list to avoid duplicates
            $Duplicate = $LTODevices | Where-Object { $_.DeviceID -eq $Device.DeviceID }
            if (-not $Duplicate) {
                $DeviceInfo = [PSCustomObject]@{
                    DeviceID = $Device.DeviceID
                    Name = $Device.Name
                    Description = $Device.Description
                    Status = $Device.Status
                    Availability = "PnP Status: OK"
                    Manufacturer = $Device.Manufacturer
                    MediaType = if ($Device.Name -match "ATTO|ExpressSAS") { "ATTO Device" } else { "PnP LTO Device" }
                    Method = "Win32_PnPEntity"
                }
                $LTODevices += $DeviceInfo
                
                if ($Device.Name -match "LTO|Tape") {
                    $DeviceFound = $true
                }
                if ($Device.Name -match "ATTO|ExpressSAS") {
                    $ATTOCardFound = $true
                }
            }
        }
        
        # Method 5: Check logical disk drives for removable media (tape drives often show here)
        Write-Host "  Checking for removable media drives..." -ForegroundColor Gray
        $RemovableDisks = Get-WmiObject -Class Win32_LogicalDisk -ErrorAction SilentlyContinue | 
            Where-Object { $_.DriveType -eq 5 } # DriveType 5 = Removable disk
        
        foreach ($Disk in $RemovableDisks) {
            if ($Disk.Description -match "Tape|LTO") {
                $DeviceInfo = [PSCustomObject]@{
                    DeviceID = $Disk.DeviceID
                    Name = $Disk.VolumeName + " (" + $Disk.DeviceID + ")"
                    Description = $Disk.Description
                    Status = "Available"
                    Availability = "Logical Drive"
                    Manufacturer = "Unknown"
                    MediaType = "Removable Tape Media"
                    Method = "Win32_LogicalDisk"
                }
                $LTODevices += $DeviceInfo
                $DeviceFound = $true
            }
        }
        
        # If we found ATTO card but no tape drives, still consider it a partial success
        if ($ATTOCardFound -and -not $DeviceFound) {
            Write-Host "  ATTO SAS card found but no LTO devices detected" -ForegroundColor Yellow
        }
        
        return @{
            Found = ($DeviceFound -or $ATTOCardFound)
            DeviceFound = $DeviceFound
            ATTOCardFound = $ATTOCardFound
            Devices = $LTODevices
            Count = $LTODevices.Count
        }
    }
    catch {
        Write-Host "Error checking for LTO devices: $($_.Exception.Message)" -ForegroundColor Red
        return @{
            Found = $false
            DeviceFound = $false
            ATTOCardFound = $false
            Devices = @()
            Count = 0
            Error = $_.Exception.Message
        }
    }
}

# Main execution
Write-Host "=== SAS LTO Device Monitor ===" -ForegroundColor Cyan
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Gray

$Result = Test-LTODevices

if ($Result.Found) {
    if ($Result.ATTOCardFound -and $Result.DeviceFound) {
        $StatusMessage = "SUCCESS: ATTO SAS card and $($Result.Count) LTO device(s) found"
        Write-Host $StatusMessage -ForegroundColor Green
    } elseif ($Result.ATTOCardFound -and -not $Result.DeviceFound) {
        $StatusMessage = "PARTIAL: ATTO SAS card found but no LTO tape drives detected"
        Write-Host $StatusMessage -ForegroundColor Yellow
    } elseif (-not $Result.ATTOCardFound -and $Result.DeviceFound) {
        $StatusMessage = "PARTIAL: LTO devices found but no ATTO SAS card detected"
        Write-Host $StatusMessage -ForegroundColor Yellow
    } else {
        $StatusMessage = "SUCCESS: Found $($Result.Count) device(s)"
        Write-Host $StatusMessage -ForegroundColor Green
    }
    
    # Display device details
    Write-Host "`nDevice Details:" -ForegroundColor Yellow
    $ATTODevices = $Result.Devices | Where-Object { $_.Method -eq "ATTO SAS Controller" -or $_.MediaType -match "ATTO" }
    $LTODevices = $Result.Devices | Where-Object { $_.Method -ne "ATTO SAS Controller" -and $_.MediaType -notmatch "ATTO" }
    
    if ($ATTODevices) {
        Write-Host "  ATTO SAS Controllers:" -ForegroundColor Cyan
        foreach ($Device in $ATTODevices) {
            Write-Host "    Name: $($Device.Name)" -ForegroundColor White
            Write-Host "    ID: $($Device.DeviceID)" -ForegroundColor Gray
            Write-Host "    Status: $($Device.Status)" -ForegroundColor Gray
            Write-Host "    ---"
        }
    }
    
    if ($LTODevices) {
        Write-Host "  LTO Tape Drives:" -ForegroundColor Cyan
        foreach ($Device in $LTODevices) {
            Write-Host "    Name: $($Device.Name)" -ForegroundColor White
            Write-Host "    ID: $($Device.DeviceID)" -ForegroundColor Gray
            Write-Host "    Status: $($Device.Status)" -ForegroundColor Gray
            Write-Host "    Type: $($Device.MediaType)" -ForegroundColor Gray
            Write-Host "    Method: $($Device.Method)" -ForegroundColor Gray
            Write-Host "    ---"
        }
    }
    
    # Send success email if requested
    if ($SendOnSuccess) {
        $EmailSubject = if ($Result.ATTOCardFound -and $Result.DeviceFound) {
            "LTO Device Monitor - ATTO SAS Card and LTO Devices Found"
        } elseif ($Result.ATTOCardFound -and -not $Result.DeviceFound) {
            "LTO Device Monitor - ATTO SAS Card Found (No LTO Devices)"
        } else {
            "LTO Device Monitor - Devices Found"
        }
        
        $EmailBody = @"
<html>
<body>
<h2>SAS LTO Device Monitor Report</h2>
<p><strong>Status:</strong> <span style="color: green;">SUCCESS</span></p>
<p><strong>Timestamp:</strong> $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')</p>
<p><strong>Total Devices Found:</strong> $($Result.Count)</p>
<p><strong>ATTO SAS Card:</strong> $(if ($Result.ATTOCardFound) { "<span style='color: green;'>Found</span>" } else { "<span style='color: orange;'>Not Found</span>" })</p>
<p><strong>LTO Tape Drives:</strong> $(if ($Result.DeviceFound) { "<span style='color: green;'>Found</span>" } else { "<span style='color: orange;'>Not Found</span>" })</p>

<h3>Device Details:</h3>
<table border="1" style="border-collapse: collapse;">
<tr style="background-color: #f0f0f0;">
<th>Device Name</th>
<th>Device Type</th>
<th>Status</th>
<th>Detection Method</th>
</tr>
"@
        foreach ($Device in $Result.Devices) {
            $DeviceType = if ($Device.MediaType -match "ATTO") { "ATTO SAS Controller" } else { "LTO Tape Drive" }
            $EmailBody += @"
<tr>
<td>$($Device.Name)</td>
<td>$DeviceType</td>
<td>$($Device.Status)</td>
<td>$($Device.Method)</td>
</tr>
"@
        }
        $EmailBody += "</table>"
        
        if ($Result.ATTOCardFound -and -not $Result.DeviceFound) {
            $EmailBody += @"
<h3>Note:</h3>
<p style="color: orange;">ATTO SAS card is present but no LTO tape drives were detected. This may indicate:</p>
<ul>
<li>LTO drive is powered off or not connected</li>
<li>SAS cable connection issue</li>
<li>LTO drive requires driver installation</li>
<li>LTO drive is in an error state</li>
</ul>
"@
        }
        
        $EmailBody += "</body></html>"
        
        $Priority = if ($Result.ATTOCardFound -and $Result.DeviceFound) { "Low" } else { "Normal" }
        Send-EmailNotification -Subject $EmailSubject -Body $EmailBody -Priority $Priority
    }
} else {
    $StatusMessage = "WARNING: No ATTO SAS card or LTO devices found"
    Write-Host $StatusMessage -ForegroundColor Red
    
    if ($Result.Error) {
        Write-Host "Error: $($Result.Error)" -ForegroundColor Red
    }
    
    # Send failure email if requested
    if ($SendOnFailure) {
        $EmailSubject = "LTO Device Monitor - No ATTO SAS Card or LTO Devices Found"
        $EmailBody = @"
<html>
<body>
<h2>SAS LTO Device Monitor Report</h2>
<p><strong>Status:</strong> <span style="color: red;">WARNING</span></p>
<p><strong>Timestamp:</strong> $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')</p>
<p><strong>Issue:</strong> No ATTO SAS card or LTO devices were detected on this system.</p>
$(if ($Result.Error) { "<p><strong>Error Details:</strong> $($Result.Error)</p>" })

<h3>What was checked:</h3>
<ul>
<li>ATTO ExpressSAS controllers via WMI</li>
<li>Win32_TapeDrive for LTO tape drives</li>
<li>SCSI controllers for LTO devices</li>
<li>Plug-and-Play devices for ATTO and LTO hardware</li>
<li>Removable media drives for tape devices</li>
</ul>

<h3>Troubleshooting Steps:</h3>
<ul>
<li><strong>ATTO SAS Card:</strong>
    <ul>
    <li>Verify the ATTO SAS card is properly seated in the PCIe slot</li>
    <li>Check that ATTO drivers are installed (download from ATTO website)</li>
    <li>Verify card appears in Device Manager</li>
    <li>Check for hardware conflicts</li>
    </ul>
</li>
<li><strong>LTO Tape Drive:</strong>
    <ul>
    <li>Verify power connection to the LTO drive</li>
    <li>Check SAS cable connections between ATTO card and LTO drive</li>
    <li>Ensure LTO drive is powered on</li>
    <li>Install proper LTO device drivers if needed</li>
    <li>Check for device errors in Event Viewer</li>
    </ul>
</li>
</ul>

<p><strong>Next Steps:</strong> Check Device Manager for any unknown devices or error states, and verify all hardware connections.</p>
</body>
</html>
"@
        
        Send-EmailNotification -Subject $EmailSubject -Body $EmailBody -Priority "High"
    }
}

Write-Host "`nScript completed at $(Get-Date -Format 'HH:mm:ss')" -ForegroundColor Gray
