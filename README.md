# Legal Hold Mattermost Plugin

> **Not recommended for production use without Mattermost guidance. Please reach out to your Customer Success Manager to learn more.**

This plugin allows administrators to place one or more users on legal hold for a set period of time.

## License

This repository is licensed under the [Mattermost Source Available License](LICENSE) and requires a valid Enterprise Edition License when used for production. See [frequently asked questions](https://docs.mattermost.com/overview/faq.html#mattermost-source-available-license) to learn more.

Although a valid Mattermost Enterprise Edition License is required if using this plugin in production, the [Mattermost Source Available License](LICENSE) allows you to compile and test this plugin in development and testing environments without a Mattermost Enterprise Edition License. As such, we welcome community contributions to this plugin.

If you're running an Enterprise Edition of Mattermost and don't already have a valid license, you can obtain a trial license from **System Console > Edition and License**. If you're running the Team Edition of Mattermost, including when you run the server directly from source, you may instead configure your server to enable both testing (`ServiceSettings.EnableTesting`) and developer mode (`ServiceSettings.EnableDeveloper`). These settings are not recommended in production environments.

## How To Install

Download the latest released version and upload to your Mattermost installation on the plugins page
of the System Console in the usual way.

## Configuring Legal Holds

Once the plugin is installed, a new "Legal Hold" section will appear in the System Console UI
in the Plugins section. There are two main settings:

* **Enable Plugin**: controls whether the plugin is enabled. It must be enabled to use it.
* **Amazon S3 Bucket Settings**: optionally use a separate S3 Bucket than the one configured for your Mattermost server.
* **Time of Day**: this setting controls at what time the delay collection of Legal Hold data
  should occur. We recommend choosing a quiet time of day to minimise impact on your users. Make
  sure to specify the time in the format shown in the example.

Below these settings is the table of Legal Holds. To create a new Legal Hold, select the
"Create legal hold" button. You must give it a name, a start date, and select at least one
user to be part of the Legal Hold. You may optionally provide a Finish Date. If you do not,
it will continue until either you do, or you release the legal hold.

Once you've created a legal hold, each day the legal hold job will run and take a copy of all
data matching the legal hold attributes. This will be stored in a separate folder in your file
storage backend. You can set a legal hold to start in the past, but only data that is still
present in your Mattermost server on the first run of the job will be saved (i.e. data that has
already been purged by a data retention policy at the time of the first run will not be included
in the legal hold). Once data is held by the Legal Hold, it will not be affected by Data Retention
policy. However, newly created Legal Holds will not be able to access data that was already purged
by Data Retention policy at the time of their first run _even if the data is held in an existing
legal hold_.

You can edit the name, end data and users in a Legal Hold. Adding new users to a legal hold will only
include their data from the next run of the hold. Similarly, removing a user from the hold will
only remove their data from the next run of the legal hold job. You can also set or change the end
date of the hold. Extending the end date of a legal hold that has already ended is allowed, but comes
with the same caveats as setting the start date above in relation to data that has already been purged.

You can download a Legal Hold data as a zip file. Remember this data is only updated once per day
by the job. You can download multiple times and it will always include all data for the entirety of
the legal hold.

You can release a legal hold. When doing this, all data within the legal hold is immediately and
permanently purged from the storage area, and cannot be recovered.

See the `processor` subdirectory for how to turn the downloaded zip file into a human readable HTML
export that you can view and search in a web browser.

## A note on downloading large legal holds

For large legal holds, the download process can take more time than the HTTP request timeout. If you are experiencing timeouts, you can increase the timeout under **System Console** > **Web server** > **Write timeout** or in your `config.json` file. This is a global setting for the entire server.

Keep in mind that the same applies for reverse proxies, which may have their own timeout settings. If you are using a reverse proxy, you may need to adjust the timeout settings there as well.

## Testing

In order to run the plugin's scheduled job on demand for testing, you can send a request to the `/api/v1/legalhold/run` endpoint, as explained in this pull request: https://github.com/mattermost/mattermost-plugin-legal-hold/pull/43
