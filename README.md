# paint-board

### A simple paint board webserver.
This program is licensed under GNU AGPLv3.

### Dependencies
golang.org/x/net/websocket [ARCHIVED]

## CLI

- -L Print license and exit.
- -D Enable debug mode.
- -c Declare max challenges.
- -m Declare memory limit.
- -w Declare WebSocket service port.
- -p Declare HTTP RESTful service port.
- -d Declare tick timestamp duration.

# Paintboard API Specification

## Roles

### Master Token Owner & Master
The maintainer. The master can create zones and operates process.

### Privileged Token Owner & Administrator
No freeze duration. Able to generate and expire limited tokens.

### Limited Token Owner
Token will be frozen for a while after a operation.

## HTTP RESTful APIs

### GET Form

**/draw**: Draw a pixel.
- **&zone=**_int_>=0: Zone ID
- **&x=**_int_>=0: X-axis
- **&y=**_int_>=0: Y-axis
- **&r=,&g=,&b=,&a=**_uint8_: RGBA color value, if alpha != 0 the pixel will be ignored.

**/print**: Print entire zone. It freezes limited operator's IP for a while.
- **&zone=**_int_>=0: Zone ID
- **Reponse Body**:
  - MIME: image/png

**/gen**: Generate a limited token. **Privileged token required**.
- **&zone=**_int_>=0: Zone ID
- **Response Header**:
  - **token**: Generated token.

**/create**: Create new zone. **Master token required**.
- **&x=**_int_>0: Width
- **&y=**_int_>0: Height
- **&freeze=**_int_>=0: Freeze/cooling duration/ticks.
- **Response Header**:
  - **zone**: Zone ID
  - **token**: Privileged token.

**/stop**: Stop process. **Master token required**

Each response header contains "text" field.

### HTTP Status Codes

- 200 Request was executed successfully.
- 202 The requester IP address was recorded but the operation was not executed.
- 400 Bad request. Bad format.
- 401 Unauthorized. The token might be invalid.
- 403 Requester IP address has been banned, due to the times of challenge failures has reached the limit.
- 404 The zone required by the request was not found.
- 409 Operation conflict.
- 413 Request content is too large.

## WebSocket APIs

Pure byte stream.

Pending.