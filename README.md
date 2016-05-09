# nettee

__nettee__ is a program useful to pipe, like __tee__ and __nc__ can do together, your output over the net

```bash
$ nettee
usage: nettee [options] --sink=(udp|tcp):<address>:<port>...

  options:

    -h, --help                   exactly what you are reading now
    --source=<filename>          watch a file like 'less +F <filename>' do
    --no-output                  do not print log entries to stdin
    --show-date                  attach datetime to every log entry
    
```

### A few examples of how to use it

[![asciicast](https://asciinema.org/a/akn7p54qchzjf8vjob88j61ol.png)](https://asciinema.org/a/akn7p54qchzjf8vjob88j61ol)
