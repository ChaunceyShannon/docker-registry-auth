I have a private docker image registry server that I am using for a while, and sometimes I have some temporary servers that need to do some temporary jobs. Since the username and password are too complex that I can not remember, it needs to login to my private registry server every time for pulling images makes me sick, I decide to give anonymous access to pull images from my private registry server. 

---

After some research by monitoring the http requests, I found that no matter `push` or `pull`, the `docker` command will always check the path `/v2/` at the first time visit.

It will lead `push` to fail if the registry service enables authentication and the first check of the path `/v2/` did not return 401, because the `POST` request will not attach an authentication HTTP header, but if the server return 401, the `push` action will become succeed.

If the needed action is `pull` and the first check of the path `/v2/` returns `401`, `docker` command will notice the `pull` action needs to be authorized, but if the server returns `403`, the `docker` command will go on check the manifests of the image.

But if the action is `push` and the first check of the path `/v2/` returns 403, `docker` will notice forbidden and exit without prompt for login. 

So if the next action is `push`, I can return `401` to require authentication, and `403` for `pull` action to go on pulling the image without a credential.

I have checked the HTTP packet with Wireshark for the first check of the path `/v2/` carefully, but I did not find anything to indicate whether the next action is `push` or `pull`, otherwise I can simply achieve the goal by control the return status code.

After thinking for a while, I decide to use different domains to do the things. Provide anonymous access of the `GET` and `HEAD` method to registry server in some specific path to a public domain, and all other domains require full authentication.

---

Run the image as a container with these environment variables 

* user: username for the push action 
* pass： password for the push action 
* public_domain: Domain name that provided anonymous access of `GET` and `HEAD` method

```
docker run \
    -d \
    --name registry \
    --hostname registry \
    --restart=unless-stopped \
    -v /data/registry/images:/var/lib/registry \
    -e public_domain=public.example.com \
    -e user=username \
    -e pass=password \
    -e REGISTRY_STORAGE_DELETE_ENABLED=true \
    chaunceyshannon/registry:2
```

The application will first start an official registry server with the default config file normally, and then listen on TCP 5001 port and forward the request to 5000 port that the official registry server is listened on. 

All the things the reverse proxy needs to do is to forward traffic to port 5001 of the container, no matter what domain is.
