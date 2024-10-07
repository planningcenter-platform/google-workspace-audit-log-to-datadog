# Google Workspace Audit Log to Datadog

Alternative to the [Datadog Google Workspace integration](https://docs.datadoghq.com/integrations/gsuite/).

### Why?

As of October 2024, the Datadog Google Workspace integration imports logs on a 90 minute delay. From their documentation:

> Note: The Groups, Enterprise Groups, Login, Token, and Calendar logs are on a 90 minute crawler because of a limitation in how often Google polls these logs on their side. Logs from this integration are only forwarded every 1.5-2 hours.

While it is true that this is somewhat inevitable due to Google Audit Logs having a [lag time](https://support.google.com/a/answer/7061566?hl=en), _some_ logs do show up sooner. When building security alerts it is preferable to have the most up-to-date logs available.

Additionally, and more importantly, **the current Datadog integration has a tendency to miss some `login` event logs**, as those have a lag time that exceeds the 90 minute delay they operate on. In practice, between 30 and 40% of our `login` events were not imported into Datadog.

### How?

This pulls all of the audit logs for each service with a time window at least 2x the [lag time](https://support.google.com/a/answer/7061566?hl=en) for that service. Each event in the logs is then stored on S3 with the key name of the `etag` for the event.

Each time a new file is created on S3, the contents of that file is sent to Datadog as a log entry. By only running this trigger on _create_ we are using S3 to deduplicate the logs.

Files in S3 are cleaned up with a Lifecycle policy that removes them after 7 days.

**Why not use [notifications](https://developers.google.com/admin-sdk/reports/reference/rest/v1/activities/watch)?**
The [documentation](https://developers.google.com/admin-sdk/reports/v1/guides/push#understand-admin-sdk-api-notification-events) states that we should not rely solely on notifications. Since we have to poll to ensure data integrity, we might as well _only_ poll.

> Notifications are not 100% reliable. Expect a small percentage of messages to get dropped under normal working conditions. Make sure to handle these missing messages gracefully, so that the application still syncs even if no push messages are received.

## Setup

This template should be deployable to your AWS account with no issues. You will need to set these parameters:

- `GoogleCredentialsParameter`
- `GoogleAdminEmailParameter`
- `DatadogApiKeyParameter`

### `GoogleCredentialsParameter`

1. Open your [GCP dashboard](https://console.cloud.google.com/apis/dashboard) and create a new project, ex `Google Audit Log to Datadog`
1. Go to _API & Services_ and click _+Enable APIs and Services_ in the top.
1. Search for _Admin SDK_ and _Enable_ the API.
1. Follow [this guide](https://developers.google.com/workspace/guides/create-credentials#service-account) from Google and:

- Create a Service Account
- Create credentials for a service account (**The contents of this `.json` file will be the value of this parameter**)
- Set up domain-wide delegation for a service account
  - Add the `https://www.googleapis.com/auth/admin.reports.audit.readonly` scope

### `GoogleAdminEmailParameter`

Email address of a Google Workspace user with permissions to access the admin reports.

### `DatadogApiKeyParameter`

_Organization Settings_ > _API Keys_ > _New Key_.
