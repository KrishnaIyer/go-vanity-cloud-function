# Go Vanity Cloud Function

This is a simple Google Cloud Function written in Go that creates an HTTP request handler that handles Go Vanity import redirects. 

The code is based off the [GCP example repository](https://github.com/GoogleCloudPlatform/govanityurls).

## Motivation

Vanity redirection is as simple as an HTTP redirect and I don't think deploying a server or even a container is necessary. This is one of the purest use cases for Cloud Functions.

## Design Choices

## Testing

A local testing framework is included. In order to test the redirection meta tags locally, set the `LOCAL_PORT` env value and `localhost:$LOCAL_PORT` will be used as the `host` value.

Now you can query `http://localhost:$LOCAL_PORT`.

Ex: Set `DEBUG_ADDRESS=localhost:8080`

```bash
curl -v http://localhost:8080 #index
curl -v http://localhost:8080/reponame #for a repo
```

> Note: You still need to provide a valid remote configuration file which is read and the `host` value is overwritten by `localhost:$LOCAL_PORT`.

## Deployment

To deploy this function, you need the following things
- A GCP Project (requires a GCP account).
- `gcloud` [cli](https://cloud.google.com/sdk/gcloud) setup and configured locally.


## License
