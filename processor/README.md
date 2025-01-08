Mattermost Legal Hold Processor
===============================

This tool processes exported data from the Mattermost Legal Hold plugin
into human-readable HTML pages. It's available in both command-line and GUI versions.

Command Line Usage
----------------

Build the command-line version:
```shell
go build
```

Extract your legal hold export Zip file to a directory, then run:
```shell
$ ./processor --legal-hold-data ./legal-hold-directory --output-path ./output-directory --legal-hold-secret "your secret"
```

GUI Usage
--------

Build and run the GUI version:
```shell
cd processor/gui
go build
./gui
```

The GUI allows you to:
1. Select the directory containing your extracted legal hold data
2. Choose an output directory for the processed files
3. Optionally provide a secret key
4. Process the legal hold with visual progress updates

For both versions, when processing completes, open the generated `index.html` in your browser
to browse the legal hold data. Use Ctrl+F/Cmd+F to search for specific text.
