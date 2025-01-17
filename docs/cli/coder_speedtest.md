## coder speedtest

Run upload and download tests from your machine to a workspace

```
coder speedtest <workspace> [flags]
```

### Options

```
  -d, --direct          Specifies whether to wait for a direct connection before testing speed.
  -h, --help            help for speedtest
  -r, --reverse         Specifies whether to run in reverse mode where the client receives and the server sends.
  -t, --time duration   Specifies the duration to monitor traffic. (default 5s)
```

### Options inherited from parent commands

```
      --global-config coder   Path to the global coder config directory.
                              Consumes $CODER_CONFIG_DIR (default "~/.config/coderv2")
      --header stringArray    HTTP headers added to all requests. Provide as "Key=Value".
                              Consumes $CODER_HEADER
      --no-feature-warning    Suppress warnings about unlicensed features.
                              Consumes $CODER_NO_FEATURE_WARNING
      --no-version-warning    Suppress warning when client and server versions do not match.
                              Consumes $CODER_NO_VERSION_WARNING
      --token string          Specify an authentication token. For security reasons setting CODER_SESSION_TOKEN is preferred.
                              Consumes $CODER_SESSION_TOKEN
      --url string            URL to a deployment.
                              Consumes $CODER_URL
  -v, --verbose               Enable verbose output.
                              Consumes $CODER_VERBOSE
```

### SEE ALSO

- [coder](coder.md) -
