# web-test-api

This is a simple UI application to:

- [x] Send/receive requests/responses.
- [ ] Receive data that Gateway (use EdgeX foundry) export to.

# Run

- Start MQTT Borker in your local (example: `sudo service mosquitto start`)
- Pull EdgeX: `docker-compose pull`
- Run EdgeX: `docker-compose up -d`
- Clone repository
- `cd web-test-api`
- `make run`
- Open browser, and type: `http:localhost:3333`

# Usage

1. Setup the MQTT connection configuration
2. Next, fill/select the informations: Method, Api, Body (if any), RequestID (or press Generate button)
3. Press "Send request" button
4. The response will show in below

# Author: 

*Phan Van Hai*