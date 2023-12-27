## Analyze

### A couple of examples how to parse sites use chromium-chromedriver

Before we need install driver:

```sh
sudo apt -y install chromium-chromedriver
```

### Build and start docker container

```sh
docker buildx build . -t analyze
docker run -v /dev/shm:/dev/shm -itd --rm analyze
```



