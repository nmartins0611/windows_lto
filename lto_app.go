package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	SMTPServer   string `json:"smtp_server"`
	SMTPPort     string `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	ToEmail      string `json:"to_email"`
	CheckTime    string `json:"check_time"` // Format: "07:00" for 7 AM
}

// LTOMonitor handles the monitoring logic
type LTOMonitor struct {
	config        Config
	logger        *log.Logger
	consoleLogger *log.Logger
}

func main() {
	// Parse command line flags
	testEmail := flag.Bool("test", false, "Send a test email and exit")
	createConfig := flag.Bool("setup", false, "Create a new config.json file interactively")
	smtpServer := flag.String("smtp-server", "", "SMTP server address")
	smtpPort := flag.String("smtp-port", "587", "SMTP port")
	smtpUser := flag.String("smtp-user", "", "SMTP username")
	smtpPassword := flag.String("smtp-password", "", "SMTP password")
	fromEmail := flag.String("from-email", "", "From email address")
	toEmail := flag.String("to-email", "", "Administrator email address")
	checkTime := flag.String("check-time", "08:00", "Daily check time (HH:MM format)")
	
	flag.Parse()

	// Initialize logger
	logFile, err := os.OpenFile("lto_monitor.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	
	// Also log to console for immediate feedback
	consoleLogger := log.New(os.Stdout, "", log.LstdFlags)
	
	logger.Println("=== LTO Monitor starting ===")
	consoleLogger.Println("LTO Monitor starting...")

	// If setup flag is provided, create config interactively
	if *createConfig {
		createConfigInteractively(logger, consoleLogger)
		return
	}

	// If command line args provided, create config from them
	if *smtpServer != "" || *smtpUser != "" || *toEmail != "" {
		err := createConfigFromArgs(*smtpServer, *smtpPort, *smtpUser, *smtpPassword, 
			*fromEmail, *toEmail, *checkTime, logger, consoleLogger)
		if err != nil {
			consoleLogger.Fatalf("Failed to create config: %v", err)
		}
		consoleLogger.Println("Configuration created successfully!")
		return
	}

	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
		consoleLogger.Fatalf("Failed to load configuration: %v", err)
	}

	logger.Printf("Configuration loaded successfully. Check time: %s, Admin email: %s", 
		config.CheckTime, config.ToEmail)

	monitor := &LTOMonitor{
		config:        config,
		logger:        logger,
		consoleLogger: consoleLogger,
	}

	// If test flag is provided, send test email and exit
	if *testEmail {
		monitor.sendTestEmail()
		return
	}

	// Run the monitoring loop
	monitor.run()
}

func createConfigFromArgs(smtpServer, smtpPort, smtpUser, smtpPassword, fromEmail, toEmail, checkTime string, logger, consoleLogger *log.Logger) error {
	logger.Println("Creating configuration from command line arguments")
	consoleLogger.Println("Creating configuration from command line arguments...")

	// Validate required fields
	if smtpServer == "" {
		return fmt.Errorf("smtp-server is required")
	}
	if smtpUser == "" {
		return fmt.Errorf("smtp-user is required")
	}
	if toEmail == "" {
		return fmt.Errorf("to-email is required")
	}

	// Set defaults
	if fromEmail == "" {
		fromEmail = smtpUser
	}
	if smtpPassword == "" {
		consoleLogger.Print("Enter SMTP password: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			smtpPassword = scanner.Text()
		}
	}

	config := Config{
		SMTPServer:   smtpServer,
		SMTPPort:     smtpPort,
		SMTPUser:     smtpUser,
		SMTPPassword: smtpPassword,
		FromEmail:    fromEmail,
		ToEmail:      toEmail,
		CheckTime:    checkTime,
	}

	return saveConfig(config, "config.json", logger, consoleLogger)
}

func createConfigInteractively(logger, consoleLogger *log.Logger) {
	logger.Println("Creating configuration interactively")
	consoleLogger.Println("=== LTO Monitor Configuration Setup ===")
	consoleLogger.Println("Please provide the following email server details:")

	scanner := bufio.NewScanner(os.Stdin)
	config := Config{}

	// SMTP Server
	consoleLogger.Print("SMTP Server (e.g., smtp.gmail.com): ")
	scanner.Scan()
	config.SMTPServer = strings.TrimSpace(scanner.Text())

	// SMTP Port
	consoleLogger.Print("SMTP Port [587]: ")
	scanner.Scan()
	port := strings.TrimSpace(scanner.Text())
	if port == "" {
		config.SMTPPort = "587"
	} else {
		config.SMTPPort = port
	}

	// SMTP User
	consoleLogger.Print("SMTP Username (email): ")
	scanner.Scan()
	config.SMTPUser = strings.TrimSpace(scanner.Text())

	// SMTP Password
	consoleLogger.Print("SMTP Password: ")
	scanner.Scan()
	config.SMTPPassword = strings.TrimSpace(scanner.Text())

	// From Email
	consoleLogger.Printf("From Email [%s]: ", config.SMTPUser)
	scanner.Scan()
	fromEmail := strings.TrimSpace(scanner.Text())
	if fromEmail == "" {
		config.FromEmail = config.SMTPUser
	} else {
		config.FromEmail = fromEmail
	}

	// To Email
	consoleLogger.Print("Administrator Email (notifications): ")
	scanner.Scan()
	config.ToEmail = strings.TrimSpace(scanner.Text())

	// Check Time
	consoleLogger.Print("Daily Check Time (HH:MM) [08:00]: ")
	scanner.Scan()
	checkTime := strings.TrimSpace(scanner.Text())
	if checkTime == "" {
		config.CheckTime = "08:00"
	} else {
		config.CheckTime = checkTime
	}

	// Validate required fields
	if config.SMTPServer == "" || config.SMTPUser == "" || config.SMTPPassword == "" || config.ToEmail == "" {
		consoleLogger.Println("ERROR: All fields except 'From Email' and 'Check Time' are required!")
		return
	}

	// Save configuration
	err := saveConfig(config, "config.json", logger, consoleLogger)
	if err != nil {
		consoleLogger.Printf("Failed to save configuration: %v", err)
		return
	}

	consoleLogger.Println("\n=== Configuration Summary ===")
	consoleLogger.Printf("SMTP Server: %s:%s", config.SMTPServer, config.SMTPPort)
	consoleLogger.Printf("From Email: %s", config.FromEmail)
	consoleLogger.Printf("To Email: %s", config.ToEmail)
	consoleLogger.Printf("Check Time: %s", config.CheckTime)
	consoleLogger.Println("\nConfiguration saved successfully!")
	consoleLogger.Println("You can now run the application normally or use --test to send a test email.")
}

func saveConfig(config Config, filename string, logger, consoleLogger *log.Logger) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	err = os.WriteFile(filename, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	logger.Printf("Configuration saved to %s", filename)
	return nil
}

func loadConfig(filename string) (Config, error) {
	var config Config

	// Check if config file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return config, fmt.Errorf("config file '%s' not found. Use --setup for interactive setup or provide command line arguments", filename)
	}

	// Read the config file
	data, err := os.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse JSON
	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate required fields
	if config.SMTPServer == "" || config.SMTPUser == "" || config.SMTPPassword == "" ||
		config.FromEmail == "" || config.ToEmail == "" {
		return config, fmt.Errorf("missing required configuration fields. Please check your config.json file")
	}

	// Set defaults
	if config.SMTPPort == "" {
		config.SMTPPort = "587"
	}
	if config.CheckTime == "" {
		config.CheckTime = "08:00"
	}

	return config, nil
}

func (m *LTOMonitor) run() {
	m.logger.Printf("Monitor started. Checking time: %s", m.config.CheckTime)
	m.consoleLogger.Printf("Monitor started. Daily check scheduled for: %s", m.config.CheckTime)

	for {
		now := time.Now()
		targetTime, err := time.Parse("15:04", m.config.CheckTime)
		if err != nil {
			m.logger.Printf("ERROR: Error parsing check time: %v", err)
			m.consoleLogger.Printf("ERROR: Error parsing check time: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Set target time for today
		target := time.Date(now.Year(), now.Month(), now.Day(),
			targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())

		// If target time has passed today, set it for tomorrow
		if now.After(target) {
			target = target.Add(24 * time.Hour)
		}

		// Sleep until target time
		duration := target.Sub(now)
		m.logger.Printf("Next check scheduled for: %v (sleeping for %v)", target, duration)
		m.consoleLogger.Printf("Next check scheduled for: %v (sleeping for %v)", target, duration)
		time.Sleep(duration)

		// Perform the check
		m.performCheck()

		// Sleep for a minute to avoid immediate re-execution
		time.Sleep(1 * time.Minute)
	}
}

func (m *LTOMonitor) performCheck() {
	m.logger.Println("=== Performing LTO library check ===")
	m.consoleLogger.Println("Performing LTO library check...")

	startTime := time.Now()
	connected := m.checkLTOConnection()
	duration := time.Since(startTime)
	
	if connected {
		m.logger.Printf("SUCCESS: LTO library is connected (check took %v)", duration)
		m.consoleLogger.Println("SUCCESS: LTO library is connected")
		m.sendEmail("LTO Library Status - OK", 
			fmt.Sprintf("The LTO library is connected and accessible.\n\nCheck completed in: %v", duration))
	} else {
		m.logger.Printf("FAILURE: LTO library connection failed! (check took %v)", duration)
		m.consoleLogger.Println("FAILURE: LTO library connection failed!")
		m.sendEmail("LTO Library Status - ERROR", 
			fmt.Sprintf("WARNING: The LTO library connection check failed. Please verify the connection and Atto SAS card status.\n\nCheck completed in: %v", duration))
	}
	
	m.logger.Println("=== Check completed ===")
}

func (m *LTOMonitor) checkLTOConnection() bool {
	// Method 1: Check using Windows Device Manager via PowerShell
	if m.checkDeviceManager() {
		return true
	}

	// Method 2: Check using WMI
	if m.checkWMI() {
		return true
	}

	// Method 3: Check for tape devices in system
	if m.checkTapeDevices() {
		return true
	}

	return false
}

func (m *LTOMonitor) checkDeviceManager() bool {
	m.logger.Println("Checking via Device Manager...")
	
	// PowerShell command to check for SCSI controllers and tape devices
	cmd := exec.Command("powershell", "-Command", 
		"Get-WmiObject -Class Win32_SCSIController | Where-Object {$_.Name -like '*Atto*' -or $_.Name -like '*SAS*'} | Select-Object Name, Status")
	
	output, err := cmd.Output()
	if err != nil {
		m.logger.Printf("Device Manager check error: %v", err)
		return false
	}

	outputStr := string(output)
	m.logger.Printf("Device Manager output: %s", outputStr)
	
	// Check if Atto or SAS controller is present and OK
	result := strings.Contains(strings.ToLower(outputStr), "atto") || 
		     (strings.Contains(strings.ToLower(outputStr), "sas") && strings.Contains(strings.ToLower(outputStr), "ok"))
		     
	m.logger.Printf("Device Manager check result: %v", result)
	return result
}

func (m *LTOMonitor) checkWMI() bool {
	m.logger.Println("Checking via WMI...")
	
	// Check for tape drives
	cmd := exec.Command("powershell", "-Command", 
		"Get-WmiObject -Class Win32_TapeDrive | Select-Object Name, Status, Availability")
	
	output, err := cmd.Output()
	if err != nil {
		m.logger.Printf("WMI tape drive check error: %v", err)
	} else {
		outputStr := string(output)
		m.logger.Printf("WMI tape drive output: %s", outputStr)
		
		// Check if any tape drives are found
		if strings.Contains(strings.ToLower(outputStr), "tape") {
			m.logger.Printf("WMI tape drive check result: true")
			return true
		}
	}

	// Also check for medium changers (library)
	cmd = exec.Command("powershell", "-Command", 
		"Get-WmiObject -Class Win32_CDROMDrive | Where-Object {$_.MediaType -like '*changer*'} | Select-Object Name, Status")
	
	output, err = cmd.Output()
	if err != nil {
		m.logger.Printf("WMI medium changer check error: %v", err)
		return false
	}

	outputStr := string(output)
	m.logger.Printf("WMI medium changer output: %s", outputStr)
	
	result := strings.Contains(strings.ToLower(outputStr), "changer")
	m.logger.Printf("WMI medium changer check result: %v", result)
	return result
}

func (m *LTOMonitor) checkTapeDevices() bool {
	m.logger.Println("Checking for tape devices...")
	
	// Use the 'mt' command equivalent on Windows or check for \\.\TAPE devices
	cmd := exec.Command("powershell", "-Command", 
		"Get-ChildItem -Path '\\\\?\\' -ErrorAction SilentlyContinue | Where-Object {$_.Name -like 'TAPE*'}")
	
	output, err := cmd.Output()
	if err != nil {
		m.logger.Printf("Tape device check error: %v", err)
		return false
	}

	outputStr := string(output)
	m.logger.Printf("Tape device output: %s", outputStr)
	
	result := strings.Contains(strings.ToUpper(outputStr), "TAPE")
	m.logger.Printf("Tape device check result: %v", result)
	return result
}

func (m *LTOMonitor) sendEmail(subject, body string) {
	m.logger.Printf("=== Attempting to send email ===")
	m.logger.Printf("Subject: %s", subject)
	m.logger.Printf("To: %s", m.config.ToEmail)
	m.logger.Printf("From: %s", m.config.FromEmail)
	m.logger.Printf("SMTP Server: %s:%s", m.config.SMTPServer, m.config.SMTPPort)

	startTime := time.Now()

	// Set up authentication
	auth := smtp.PlainAuth("", m.config.SMTPUser, m.config.SMTPPassword, m.config.SMTPServer)

	// Compose message
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s\r\n"+
		"\r\n"+
		"Timestamp: %s\r\n"+
		"Server: %s\r\n",
		m.config.ToEmail,
		m.config.FromEmail,
		subject,
		body,
		time.Now().Format("2006-01-02 15:04:05"),
		getHostname()))

	// Send email
	err := smtp.SendMail(m.config.SMTPServer+":"+m.config.SMTPPort, auth,
		m.config.FromEmail, []string{m.config.ToEmail}, msg)

	duration := time.Since(startTime)

	if err != nil {
		m.logger.Printf("FAILED to send email (took %v): %v", duration, err)
		m.consoleLogger.Printf("FAILED to send email: %v", err)
	} else {
		m.logger.Printf("SUCCESS: Email sent successfully (took %v)", duration)
		m.consoleLogger.Printf("SUCCESS: Email sent to %s", m.config.ToEmail)
	}
	
	m.logger.Printf("=== Email send attempt completed ===")
}

func (m *LTOMonitor) sendTestEmail() {
	m.logger.Println("=== Sending test email ===")
	m.consoleLogger.Println("Sending test email...")
	
	subject := "LTO Monitor - Test Email"
	body := fmt.Sprintf(`This is a test email from the LTO Monitor application.

If you receive this email, the email configuration is working correctly.

Configuration Details:
- SMTP Server: %s:%s
- From Email: %s
- To Email: %s
- Check Time: %s

The application is ready to monitor your LTO library.`, 
		m.config.SMTPServer, 
		m.config.SMTPPort,
		m.config.FromEmail,
		m.config.ToEmail,
		m.config.CheckTime)
	
	m.sendEmail(subject, body)
	
	m.logger.Println("=== Test email completed ===")
	m.consoleLogger.Println("Test email completed. Check the log file for details.")
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "Unknown"
	}
	return hostname
}
