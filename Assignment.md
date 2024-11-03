## Overview

We are providing skeleton code for a Golang service that resizes images. The code for the service is kept basic intentionally, and you are encouraged to make design and implementation changes along the way. You can assume a single-instance deployment for your implementation. A more elaborate production design can be discussed in a later session.

### Service Endpoints

The service currently has two endpoints:

- **`/v1/resize`** – A `POST` endpoint that takes image URLs and dimensions as input and returns cached resized images.
- **`/v1/image/{image hash}.jpg`** – An endpoint that takes an image hash and returns a previously resized image.

### Sample Input for `/v1/resize` Endpoint

```json
{
  "urls": [
    "https://i.imgur.com/RzW6QSI.jpeg",
    "https://i.imgur.com/JT7SP0M.jpeg"
  ],
  "width": 200,
  "height": 300
}
```

### Sample Output

```json
[
  {
    "result": "success",
    "url": "http://serverhost/v1/image/MqVdTC05lsQEVr0aEKAzS_2_G8rUdhxELHSi1BT8Uu8=.jpeg",
    "cached": true
  },
  {
    "result": "success",
    "url": "http://serverhost/v1/image/m29pHG80oTmWFOyBo0-EzokQ_Z_IlZ7YetxCGWXUJME=.jpeg",
    "cached": false
  }
]
```

## Assignment: Blocking vs Non-blocking calls
Do an extension of the image resizing service to support two new versions of the resize call:
- One to return a result URL immediately (do resizing asynchronously in the background)
/v1/resize?async=true
- Another to block and return once the underlying resize operation completes (default option)
/v1/resize?async=false
If there is no image in cache but the file is being processed, the existing image endpoint /v1/image/ should block and wait until the file is processed and return the image when done.
It would be nice to implement timeouts on all these potentially blocking calls!



##  Timing
Our expectation is that you spend 2 - 4 hours on this assignment. We know that is a good chunk of time and we appreciate you doing this assignment! If you're (re)learning Go, you may need a bit more time. However, any additional time spent on the assignment is at your discretion and not expected or required. This code will not be used in a production environment, so feel free to take shortcuts as needed.