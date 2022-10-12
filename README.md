# Traefik Maintenance Plugin by Programic

Traefik maintenance plugin to show visitors a maintenance page. This plugin can be used for example when upgrading a production environment.

## Configuration

The `test` directory shows a fully working setup for using this plugin in combination with Traefik in Docker. You can use this configuration for your own project. Below an explanation.

The following declaration (given here in YAML) defines a plugin:

```yaml
# Static configuration
# File: test/services/traefik/traefik.yml

experimental:
  localPlugins:
    maintenance:
      moduleName: github.com/programic/traefik-maintenance-plugin

```

Here is an example of a file provider dynamic configuration (given here in YAML), where the interesting part is the http.middlewares section:

```yaml
# Dynamic configuration
# File: test/services/traefik/dynamic/middleware.yml

http:
  middlewares:
    maintenance: # Middleware name
      plugin:
        maintenance: # Plugin name
          informUrl: "http://inform/inform.json"
          informInterval: 5
          informTimeout: 3
```

### Properties

- `informUrl` (required): Url to the `inform.json` to check if hosts are under maintenance. In directory `test/services/inform/inform.json` is an example.
- `informInterval` (optional): Every how many seconds should the inform url be consulted.
- `informTimeout` (optional): The timeout of the inform url.

## Local development

To test this plugin for local development, you can do the following.

### Prerequisites

- `docker` and `docker-compose` installed.
- `*.test` refers to your local development environment through `dnsmasq`.

### 1. Start the local development environment

```bash
$ cd test
$ docker-compose up -d
```

### 2. Open your browser

1. Go to [maintenance.test](http://maintenance.test). 
2. You will now see a maintenance page or a welcome page.
3. Change the ip in file `test/services/inform/inform.json` to your local Docker ip address.
4. Go back to the [maintenance.test](http://maintenance.test) and see what happened.

### 3. Happy coding!

You can now edit the plugin locally. Don't forget to restart Docker every time:

```bash
$ docker-compose down && docker-compose up -d
```