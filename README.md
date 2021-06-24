# antispam

Remove spam from your IMAP account.

Do you get spam? If you don't use a spam filter (or your service provider doesn't provide one), then this can be a daily annoyance of deleting tens, hundreds, or thousands of nuisance spam emails in your inbox. Ain't nobody got time for that.

`antispam` is a little binary that reads in messages from your inbox and deletes the ones that are spam. Nice, eh?

How does it do this? Pretty simple: domain & email blocklists. Three massive blocklists are included in the binary to identify spam From addresses.

If you notice someone not on this list, you can add it to your configuration. See below.

## Installation

```console
$ go get -u github.com/parkr/antispam
```

## Configuring

Configuration is via a JSON file. It has 6 possible fields, but only 4 are required:

```json
{
  "Address": "mail.example.com",
  "Port": "993",
  "Username": "email@example.com",
  "Password": "myplaintextpassword"
}
```

That will log into `mail.example.com:993` as `email@example.com` with password `myplaintextpassword`. Easy!

Two optional configuration options are `BadEmailDomains` and `BadEmails`.

```json
{
  ...
  "BadEmailDomains": ["horriblehepsebah.com", "iwillspamyou.biz"],
  "BadEmails": ["somenastyspammer@gmail.com", "someonewhoseaccountwascompromised@verizon.net"]
}
```

`BadEmailDomains` tells `antispam` to delete any email from an email address at that domain.

`BadEmails` tells `antispam` to delete any email from any of the given email addresses.

## Usage

```console
$ antispam -config=path/to/config.json -num=50
```
