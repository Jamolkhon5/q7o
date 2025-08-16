package email

const (
	VerificationEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 10px; padding: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 32px; font-weight: bold; color: #4CAF50; }
        .code-box { background: #f8f8f8; border: 2px dashed #4CAF50; border-radius: 8px; padding: 20px; text-align: center; margin: 30px 0; }
        .code { font-size: 36px; font-weight: bold; color: #333; letter-spacing: 8px; }
        .footer { text-align: center; color: #666; font-size: 12px; margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">Q7O</div>
            <h2>Welcome, {{.Username}}!</h2>
        </div>
        <p>Thank you for registering with Q7O. To complete your registration, please enter the verification code below:</p>
        <div class="code-box">
            <div class="code">{{.Code}}</div>
        </div>
        <p>This code will expire in 15 minutes.</p>
        <p>If you didn't create an account, please ignore this email.</p>
        <div class="footer">
            <p>&copy; 2024 Q7O. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

	MissedCallEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; background-color: #f4f4f4; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: white; border-radius: 10px; padding: 30px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 32px; font-weight: bold; color: #4CAF50; }
        .call-info { background: #fff3cd; border-left: 4px solid #ffc107; padding: 15px; margin: 20px 0; }
        .button { display: inline-block; background: #4CAF50; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; margin-top: 20px; }
        .footer { text-align: center; color: #666; font-size: 12px; margin-top: 30px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">Q7O</div>
            <h2>Missed Call</h2>
        </div>
        <div class="call-info">
            <p><strong>{{.CallerName}}</strong> tried to call you at {{.Time}}</p>
        </div>
        <p>Log in to Q7O to call them back!</p>
        <center>
            <a href="{{.AppURL}}" class="button">Open Q7O</a>
        </center>
        <div class="footer">
            <p>&copy; 2024 Q7O. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`
)
