# Secret Service
Ever wanted to share a file or information via a link that only works once and didn't want to have to pay or sign in to something to do so?
Then this is the service for you.

You can self host this service behind whatever reverse proxy you prefer.
Run once to generate the config in the same folder as application.
Or
Use the following environmental variables:
- server_name
- server_port
- server_domain
- server_apikey
- database_name
- database_key
- captcha_enabled
- captcha_sitekey
- captcha_secretkey
- captcha_score

This service was made over 2 days for fun and is getting the rough edges worked on to get it to version 1.
To Do:
- [ ] Embed web folder
- [ ] Finalize CSS
- [ ] File encryption
- [X] DB encryption
- [X] ENV variable support
- [ ] Auto Container Release
