+++
date = "2016-01-25T22:05:08-08:00"
description = "Finding a CSRF vulnerability in phpBB"
keywords = ["phpBB", "csrf", "bbcode"]
title = "Finding a CSRF vulnerability in phpBB"
draft = false
+++

The phpBB team released phpBB version [3.1.7-PL1](https://www.phpbb.com/support/documents.php?mode=changelog#v317)
on Jan 11, 2016 which fixed a CSRF issue I found in the admin control panel BBCode
creation form. Since BBCode is basically whitelisted HTML created by admins this
CSRF vulnerability could allow an attacker to inject arbitrary HTML or JavaScript
into forum posts.

This was my first time looking at phpBB and I was very happy with actually being
able to find something with significant impact within a few hours.

## Finding a target

A good starting point for understanding the features and flow of phpBB controllers
would be to look at something users would have access to with complex operations
going on. `./phpbb/phpBB/posting.php` is a decent starting point since this is the page
users hit when creating topics/posts. This controller should have permission checks,
record creation, form submission, and maybe some HTML escaping. At the top of
the file we can see something interesting:

```php
// Grab only parameters needed here
$post_id    = request_var('p', 0);
$topic_id   = request_var('t', 0);
$forum_id   = request_var('f', 0);
$draft_id   = request_var('d', 0);
$lastclick  = request_var('lastclick', 0);

$preview    = (isset($_POST['preview'])) ? true : false;
$save       = (isset($_POST['save'])) ? true : false;
$load       = (isset($_POST['load'])) ? true : false;
$confirm    = $request->is_set_post('confirm');
$cancel     = (isset($_POST['cancel']) && !isset($_POST['save'])) ? true : false;
```

Mostly all of the request variables used in this controller are defined a the top
of the file, some of which are taken from this weird `request_var()` function
which IntelliJ tells me is deprecated. This function is basically a wrapper for
`\phpbb\request\request_interface::variable()`. Looking at `\phpbb\request\request::variable()`
it can be seen that this method returns the  requested var from some associative array.
The array is the concatenation of the `$_POST` and `$_GET` global variables. For those of you
not familiar with PHP these are globals which contain the POST request
and query vars (respectively). All of these variables are also `trim()`'d and
type casted to match the type of whatever the default value is.

This is a key bit of information: if we can find some place in a form where a
POST request is made we should also be able to make the request with a GET. Maybe
we should start looking for CSRF bugs?

## How CSRF tokens work in phpBB

Doing a simple find in file for "form" in IDEA I came across this line:

```php
if ($submit && check_form_key('posting'))
```

Looking into the `check_form_key()` function it's clear that this is the function
to check the CSRF token (using `===` mind you)... but it's done manually. And
further *down* the file:

```php
add_form_key('posting');
```

So adding and checking the CSRF token is done manually. This smells like something
that can lead to errors!

## Finding CSRF bugs! 

Let's search the project for `add_form_key()` and `check_form_key()`. What we're
looking for is files that show up in the result for `add_form_key()` but not for
`check_form_key()`.

The following image is the result of the search for `add_form_key()` and `check_form_key()`
respectively, with the admin control panel includes folder expanded:

![add_form_key(left) vs check_form_key(right)](/img/csrf_diff.png)

The greater number of `check_form_key()` calls to `add_form_key()` calls isn't
really concerning since you can check the form key as many times as you'd like.
What we're looking for is places where a form key is added but not checked.
We can see there are two places where calls to `check_form_key()` are definitely missing:

- `acp_bbcode.php`
- `acp_extensions.php`

`acp_extensions.php` isn't too interesting since that just lets admins toggle
showing unstable versions when checking extensions for updates, so the worse a
CSRF vuln here does is allow an attacker to make an admin think their extensions
are outdated.

The check in `acp_bbcode.php` is of interest though since, although the form
is submitted via POST, the `request_var()` method is used to retrieve all form
variables. We should be able to create BBCode over GET with a CSRF token omitted!

Here's a demonstration of this vulnerability:

{{< youtube 7NsUoE32cyQ >}}

## Notes

Although this vulnerability can lead to XSS, it's really not practical. By default
phpBB enforces re-authentication when admins go to the ACP and gives the admin
a different admin CP session ID. The SID also needs to be present in both the cookie
and query string by default.

In theory a timing attack is possible since session IDs are checked with the equals
operator (`$this->session_id !== $session_id`) but this is also not very practical
since sessions are tied to IP, browser, and some other information but most importantly
doing it over the network isn't exactly easy.

The only way I see this being practical is if there's also an XSS vulnerability
on some admin page that would allow you to inject a script to get the
admin SID from `document.location` then perform the exploit.

# Timeline

- July 11, 2015: Reported vulnerability 
- August 4, 2015: Received response from two different project members requesting
more information
- December 23, 2015: Followed-up with project members (never received notification
of response and I sort of forgot about this bug)
- December 23, 2015: Vendor confirmed bug
- January 11, 2016: Fix released
