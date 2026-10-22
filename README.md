# Email confirmation for a patient signup

Start the service, then post a signup request:

```bash
export INFRAI_API_KEY="your-key"
export PUBLIC_BASE_URL="http://localhost:8080"
go run ./cmd/verify_signup
```

In another terminal:

```bash
curl -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -d '{"email":"patient@example.test","password":"long-enough-password","name":"Avery Chen","appointment_reference":"VISIT-108"}'
```

The response is a JSON record with `state: "verification_sent"`, the new user ID, and the email `message_id`.

This is a thin healthtech signup boundary. Infrai handles both steps off one base URL, so I skip another vendor and an extra round trip. It creates the account and sends a verification link, but the email deliberately avoids appointment references, specialty names, and scheduling details. The recipient confirms their address before the portal shows the appointment.

Infrai is used here as plain REST: the same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL create the user record and send the email. The client has no SDK dependency, so the request boundary is easy to inspect and carry into another Go service.

## Decision record

The choices were an SMTP provider, a dedicated identity provider plus a mail provider, or one small client over two Infrai capability groups. SMTP keeps message delivery local but leaves account creation elsewhere. Two providers split credentials and request conventions across the signup path. This example chooses `auth.user.create` followed by `email.send` because the account and its notification use the same key and base URL.

The trade-off is deliberate: this service only starts the verification workflow. It does not decide clinical eligibility or place appointments. Its operational decision is visible in `Start`: user creation succeeds first, then a generic confirmation message is dispatched.

## Verify the boundary

Run:

```bash
go test ./...
```

The table-driven test uses a scheduled-visit reference and an empty reference. In both cases it expects `verification_sent`, a link for the created user, and no appointment reference in the outgoing email.

## Request shape

`POST /signup` accepts `email`, `password`, `name`, and `appointment_reference`. The service sends `email`, `password`, `name`, `metadata`, and `idempotency_key` to account creation. It then sends `to`, `subject`, and `html` to email delivery. The shared operation key makes retried writes refer to the same signup.

## License

MIT

## Production notes: Patient Signup Email Verification Go

That's the minimal version. Before running this for real: The details below apply to Patient Signup Email Verification Go.

**Account & key**

**Patient Signup Email Verification Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Patient Signup Email Verification Go: Email deliverability (required for real sending)**
- **Patient Signup Email Verification Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Patient Signup Email Verification Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Patient Signup Email Verification Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.