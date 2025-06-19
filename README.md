# LTO Library Monitor

A Windows application that monitors LTO library connections via Atto SAS cards and sends email notifications to administrators.

## Features

- **Automated LTO Detection**: Checks for LTO library connectivity using multiple methods
- **Email Notifications**: Sends status reports to administrators
- **Flexible Scheduling**: Run via Windows Task Scheduler for reliable execution
- **Comprehensive Logging**: Detailed logs for troubleshooting and audit trails
- **Easy Setup**: Interactive configuration or command-line setup

## Quick Start

Install GO on Windows

### 1. Download and Build

```batch
# Clone or download the source code
# Open Command Prompt in the project directory

# Initialize Go module and build
go mod init lto-monitor
go build -o lto-monitor.exe
```

### 2. Configure Email Settings

**Option A: Interactive Setup**
```batch
lto-monitor.exe --setup
```

**Option B: Command Line Setup**
```batch
lto-monitor.exe --smtp-server smtp.gmail.com --smtp-user yourname@gmail.com --to-email admin@company.com
```

### 3. Test Email Configuration

```batch
lto-monitor.exe --test
```

### 4. Test LTO Check

```batch
lto-monitor.exe --check
```

## Windows Task Scheduler Setup

### Method 1: Using Task Scheduler GUI

1. **Open Task Scheduler**
   - Press `Win + R`, type `taskschd.msc`, press Enter
   - Or search "Task Scheduler" in Start menu

2. **Create Basic Task**
   - Click "Create Basic Task" in the right panel
   - Name: `LTO Library Monitor - Morning`
   - Description: `Check LTO library connection and send status email`

3. **Set Trigger**
   - When: `Daily`
   - Start date: Today's date
   - Start time: `08:00:00` (8:00 AM)
   - Recur every: `1 days`
   - Click Next

4. **Set Action**
   - Action: `Start a program`
   - Program/script: `C:\path\to\lto-monitor.exe`
   - Add arguments: `--check`
   - Start in: `C:\path\to\` (directory containing the exe)
   - Click Next

5. **Create Second Task for Evening**
   - Repeat steps 2-4 with:
   - Name: `LTO Library Monitor - Evening`
   - Start time: `18:00:00` (6:00 PM)

### Method 2: Using Command Line (schtasks)

**Create Morning Task (8:00 AM)**
```batch
schtasks /create /tn "LTO Library Monitor - Morning" /tr "C:\path\to\lto-monitor.exe --check" /sc daily /st 08:00 /ru SYSTEM
```

**Create Evening Task (6:00 PM)**
```batch
schtasks /create /tn "LTO Library Monitor - Evening" /tr "C:\path\to\lto-monitor.exe --check" /sc daily /st 18:00 /ru SYSTEM
```

### Method 3: PowerShell Script

Save this as `setup-tasks.ps1` and run as Administrator:

```powershell
# Set the path to your executable
$ExePath = "C:\path\to\lto-monitor.exe"
$WorkingDir = "C:\path\to\"

# Create morning task (8:00 AM)
$Action1 = New-ScheduledTaskAction -Execute $ExePath -Argument "--check" -WorkingDirectory $WorkingDir
$Trigger1 = New-ScheduledTaskTrigger -Daily -At "08:00"
$Settings1 = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName "LTO Library Monitor - Morning" -Action $Action1 -Trigger $Trigger1 -Settings $Settings1 -User "SYSTEM"

# Create evening task (6:00 PM)
$Action2 = New-ScheduledTaskAction -Execute $ExePath -Argument "--check" -WorkingDirectory $WorkingDir
$Trigger2 = New-ScheduledTaskTrigger -Daily -At "18:00"
$Settings2 = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName "LTO Library Monitor - Evening" -Action $Action2 -Trigger $Trigger2 -Settings $Settings2 -User "SYSTEM"

Write-Host "Tasks created successfully!"
```

Run with:
```batch
powershell -ExecutionPolicy Bypass -File setup-tasks.ps1
```

## Application Usage

### Command Line Options

```
lto-monitor.exe [OPTIONS]

Options:
  --setup                    Interactive configuration setup
  --test                     Send test email and exit
  --check                    Run LTO check once and exit (for scheduled tasks)
  --smtp-server string       SMTP server address
  --smtp-port string         SMTP port (default "587")
  --smtp-user string         SMTP username
  --smtp-password string     SMTP password
  --from-email string        From email address
  --to-email string          Administrator email address
  --check-time string        Daily check time in HH:MM format (default "08:00")
```

### Usage Examples

**Setup Configuration**
```batch
# Interactive setup
lto-monitor.exe --setup

# Command line setup
lto-monitor.exe --smtp-server smtp.gmail.com --smtp-user backup@company.com --to-email admin@company.com

# Office 365 setup
lto-monitor.exe --smtp-server smtp.office365.com --smtp-port 587 --smtp-user backup@company.com --to-email it-team@company.com
```

**Testing**
```batch
# Test email configuration
lto-monitor.exe --test

# Test LTO check (scheduled task mode)
lto-monitor.exe --check

# Run continuous monitoring (legacy mode)
lto-monitor.exe
```

## Email Server Configuration

### Gmail Setup
1. Enable 2-Factor Authentication
2. Generate App Password: Google Account → Security → App passwords
3. Use app password (not your regular password)

**Configuration:**
- SMTP Server: `smtp.gmail.com`
- Port: `587`
- Username: Your Gmail address
- Password: App-specific password

### Office 365 Setup
**Configuration:**
- SMTP Server: `smtp.office365.com`
- Port: `587`
- Username: Your Office 365 email
- Password: Your Office 365 password

### Other Email Providers
| Provider | SMTP Server | Port |
|----------|-------------|------|
| Yahoo | smtp.mail.yahoo.com | 587 |
| Outlook.com | smtp-mail.outlook.com | 587 |
| iCloud | smtp.mail.me.com | 587 |

## File Structure

After setup, your directory should contain:
```
lto-monitor.exe        # Main executable
config.json           # Email configuration
lto_monitor.log       # Application logs
README.md             # This file
```

## Monitoring and Maintenance

### Check Task Status
```batch
# List all LTO monitor tasks
schtasks /query /tn "*LTO Library Monitor*"

# Check last run time and results
schtasks /query /tn "LTO Library Monitor - Morning" /fo table /v
```

### View Logs
- Check `lto_monitor.log` for detailed execution logs
- Windows Event Viewer → Task Scheduler logs
- Task Scheduler → Task History



The application provides detailed logging to help diagnose any issues with LTO detection or email delivery.
