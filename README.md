# Go Vanity Cloud Function

This is a simple Google Cloud Function written in Go that creates an HTTP request handler that handles Go Vanity import redirects. 

The code is based off the [GCP example repository](https://github.com/GoogleCloudPlatform/govanityurls).

## Motivation

Vanity redirection is as simple as an HTTP redirect and I don't think deploying a server or even a container is necessary. This is one of the purest use cases for Cloud Functions.

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

After logging in, deploy the function using

```bash
$ gcloud functions deploy go-vanity-redirection --entry-point HandleImport --runtime go113 --trigger-http --memory 128MB --env-vars-file=./env.yml --allow-unauthenticated
```

Without the `--allow-unauthenticated` flag, the caller needs to provide some form of auth to call the URL. But since we want the vanity redirection to be public, we skip this.

### Custom DNS

To use Custom DNS (ex: `go.example.com`), there needs to be a proxy/load-balancer in front of the cloud function.

The easiest and cheapest (free) option is to setup Firebase hosting.
* Go to the [Firebase Console](https://console.firebase.google.com/u/1/) and setup a new free project (the `Spark` plan).
* Install the firebase CLI and login, the instructions of which can be found in the console.
* Create a dummy directory (ex: `firebase`) and initialize firebase.

```bash
$ cd <directory>
$ firebase init
```

* In the init options
   * Select "Hosting: Configure and deploy Firebase Hosting sites".
   * Either use your project that you selected above or create a new project.
   * Leave "What do you want to use as your public directory?" that default.
   * Select `N` for "Configure as a single-page app (rewrite all urls to /index.html)?".

* Remove the `public` folder (as we are not serving any static assets).
* Replace the `firebase.json` with the following contents

```json
{
  "hosting": {
    "public":"public",  #dummy entry, but mandatory
    "redirects": [{
      "source": "/",
      "destination": "https://<location-project-name>.cloudfunctions.net/<function-name>",
      "type":302
    }
    ]
  }
}
```
> You can checkin the above file and the `firebase.rc` file to a VCS to update and redeploy the project.

Once deployed, you check that the redirection works by going to the default Firebase App URL which is of the form `firebase-project-xxxxx.web.app`.
`xxxxx` is a random 5 digit code assigned to your project by Firebase.

If this works correctly, then follow the **Add Custom Domain** guide in the Firebase Console to setup your custom domain.

### Metrics

Runtime metrics for the cloud function are available in the GCP Console.

Similarly, Firebase hosting metrics are available in the Firebase Console.

## License

The contents of this repository are released as is under the terms of the [Apache 2.0 License](LICENSE).
