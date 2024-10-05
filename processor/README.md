Mattermost Legal Hold Processor
===============================

This command line tool processes exported data from the Mattermost Legal Hold plugin
into human-readable HTML pages.

You can build it by running `go build`.

Then download your legal hold export Zip file,
and invoke the command as follows:

```shell
$ ./processor --legal-hold-data ./legalholddata.zip --output-path ./path/to/where/you/want/the/html/output --legal-hold-secret "your secret"
```

At the end, it'll print out a link to the `index.html` page.
Open that link in your browser and you can browse the legal
hold data in human-readable form. Use Ctrl+F in your
browser to search for particular text strings.
