# Privacy Policy

**Last Updated:** July 7, 2025

## Stub Copy for Development Only!

**This is not the official privacy policy. It is included for local testing and layout purposes only.  
Do not use this version in production. The official public instance of Eko, hosted at eko.kyren.codes, has its own up-to-date privacy policy.  
This stub may be based on a prior version, but may differ from the official policy in wording or scope.**

This Privacy Policy explains how we handle data when you use our anonymous communication platform. The service operates without requiring user‑submitted personal identifiers such as names or email addresses. However, we do process certain technical data (e.g. IP addresses) which may be considered personal data under some laws.

## 1. What We Collect

When using the service, we collect the following:

- **IP address and port** – for connection tracking and abuse prevention
- **User ID** – a stable 64-bit identifier generated on first login
- **Request data** – including message contents and other client requests
- **Usage metrics** – such as API usage frequency and session durations
- **System analytics** (optional) – OS, architecture, RAM, and terminal-related environment variables

System analytics are enabled by default and can be disabled at any time in the client's local configuration file. They are used solely for technical improvement and never for tracking or profiling. To reduce duplicates, the client may generate a random, local device ID not tied to your account. This ID is used only for aggregate analytics and may appear in short-term logs (see Section 3: Log Retention).

We do not require or collect real names, emails, or similar personal identifiers. If you voluntarily include such data in messages or display names, you do so at your own discretion and are responsible for its content.

## 2. Why We Collect It

We collect this data in order to:

- Maintain the functionality and stability of the service
- Debug issues and monitor abuse
- Improve system performance and reliability
- Analyze aggregate usage patterns
- Understand user environments to improve compatibility and experience

We do **not sell your data to advertisers or third parties**.

## 3. Log Retention

We store logs of requests and connection events, which may include IP addresses, User IDs, and full request content (such as messages or device analytics).

### How Logging Works

- There is always one active log file ("current"), which is written to during runtime.
- The log is rotated either when it exceeds 100MB or every midnight (UTC time) if the server isn't down.
- Each time rotation occurs, the system checks for and deletes any archived logs that are older than 7 days from their rotation timestamp.

**Note:** Logs are typically deleted after approximately 7 to 14 days, but may remain on disk longer if log rotation has not occurred (for example, due to server downtime or low activity).

## 4. Account and Message Deletion

You may delete your account at any time. When this happens:

- Your account is flagged as deleted
- All servers ("networks") **you own** are deleted, along with all messages in those servers ("networks")

Messages you’ve sent to other servers ("networks") or users ("signals") **are not deleted automatically**, but you may delete them individually if you still have access to them.

Log data associated with your account is retained temporarily as part of standard rotation, even after account deletion.

## 5. User Rights

Under GDPR and similar laws, you can:

- **Access or correct** your data
- **Erase** your account (logs auto‑delete after approximately 7 to 14 days)
- **Disable** system analytics (connection logs remain under legitimate interest)

To exercise any right, email **eko-app@protonmail.com** with your User ID.
We’ll make reasonable efforts to respond to verifiable data‑subject requests within 30 days.

## 6. Changes to This Policy

We may modify this Privacy Policy at any time, with or without notice. Your continued use of the service after any change constitutes your acceptance of the updated policy.
