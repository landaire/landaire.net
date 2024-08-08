+++
title = "MITMing the Xbox 360 Dashboard for Fun and RCE"
description = "The golden era of man-in-the-middle attacks"
summary = "The golden era of man-in-the-middle attacks"
template = "toc_page.html"
toc = true
date = "2024-07-30"

[extra]
image = "/img/xbox-dashboard/epix-mirrored.jpg"
image_width = 720
image_height = 480
hidden = false
pretext = """
The golden era of man-in-the-middle attacks
"""
+++

In the late 2000s and early 2010s my friends and I were living and breathing Xbox hacking. We were heavily interested in game betas, internal tools, and in general exploring everything the console had to offer.

In 2010 -- or maybe 2011? the dates are getting blurry to me -- my friend Emma ([@carrot_c4k3](https://twitter.com/carrot_c4k3)) was reverse engineering how the Xbox 360 dashboard worked in order to get Hulu Plus on her retail console (there are a lot more details here missing, but that's the _why_).

Somewhere along the way she discovered that there were different URLs and paths which served different "dashboard channels". A normal console would load the following endpoint as the root manifest for content:

```
http://epix.xbox.com/epix/en-US/homepage.xml
```

And alternate channels for beta audiences existed at URLs like:

```
http://epix.xbox.com/beta/preview_green/epix/en-US/homepage.xml
http://epix.xbox.com/beta/takehome_green/epix/en-US/homepage.xml
```

These manifest files contain the metadata for dynamic marketplace slots and channels that were used to serve ads:

```xml
<channel>
    <id>XBOX360</id>
    <definitionpath>epix://xbox360channel.xml</definitionpath>
  </channel>
  <channel>
    <id>XBOX_PRE_RELEASE</id>
    <channeldef>
      <description>Xbox Pre-Release</description>
      <type>online</type>
      <slot>
        <name>Marker Scene</name>
        <description>Xbox Beta Program</description>
        <description2>Preview LIVE update</description2>
        <rating>267242991</rating>
        <shallowimg>http://epix.xbox.com/shaXam/0201/df/e8/dfe8c92a-84b2-4fbc-8ee3-7af37dee567d.JPG?v=1#Beta_Audiences.JPG</shallowimg>
      </slot>
      <slot>
        <name>Beta Announcements</name>
        <description>Announcements</description>
        <description2>Read the latest information</description2>
        <rating>267242991</rating>
        <shallowimg>http://epix.xbox.com/shaXam/0201/37/6b/376b27a2-635b-4284-8fb4-713be4c98f60.JPG?v=1#Beta_Announcments.JPG</shallowimg>
        <epixid>46f2016d-0bd6-4d0b-8cb1-f2a81356b246</epixid>
        <onclick>
          <button>A</button>
          <helptext>Select</helptext>
          <action>KeyDown</action>
        </onclick>
      </slot>
      <epix>
        <id>46f2016d-0bd6-4d0b-8cb1-f2a81356b246</id>
        <format>LUAXZP</format>
        <path>http://epix.xbox.com/shaXam/0204/79/35/7935844a-91a8-45fe-a9c0-94dfa4d6c053.lzp?v=11#Beta_Announcements.lzp</path>
        <param>url=http://live:11/xedl/BetaChannelXml/external/announcements.xml</param>
      </epix>
```

The `preview_green` URL was for users in the public Xbox LIVE preview, which Emma was a part of and therefore stood out. Through examining the Lua scripts and other manifest files she discovered a reference to `epix-preview.xbox.com` and tried loading the `preview_green` path from that domain instead.

She discovered that the manifest from this domain contained _all_ of the possible dashboard channels that were being tested for the various audiences.

Emma had just discovered some critical info that **there were other beta dashboard channels**, the content of the channels reflected things that employees would typically only see, and there were multiple variations of beta channels.

## Inventing Man-in-the-Middle Attacks

A year or two before all of this, Emma and I self-discovered what a [man-in-the-middle](https://en.wikipedia.org/wiki/Man-in-the-middle_attack) (MITM) attack was in the most janky way possible. I had learned that you could share your ethernet adapter on your PC with your Xbox to tunnel your Xbox's traffic through the PC and frequently used this technique to analyze the plaintext HTTP API calls the Xbox made to the marketplace services.

I had also learned about what a [`hosts` file](https://en.wikipedia.org/wiki/Hosts_(file)) was, and one day decided to try something wild:

Redirect the ProdNet (retail) Xbox LIVE marketplace to the PartnerNet (developer) Xbox LIVE marketplace by adding an entry for marketplace.xboxlive.com in my hosts file, pointing at the PartnerNet IP address.

It didn't work for me for some reason. But Emma tried it and it worked! She was seeing content from the developer network on her retail console.

We knew how to reverse engineer PowerPC but knew absolutely nothing about networking. At this time we were loosely familiar with the idea of a MITM attack but had no idea how to actually perform one if you control the network. We walked away feeling as if we'd just discovered electricity.

## Practical MITM Attacks

Being the smart but dumb hackers we were, we decided to try using our hosts file/networking sharing trick on something even _crazier_:

We were going to mirror the  `http://epix-preview.xbox.com/epix/en-US/homepage.xml` manifest and all of its dependencies on our local machine, then set up a web server that served the content.

Ditto with `http://epix.xbox.com/beta/preview_green/epix/en-US/homepage.xml` URLs -- we would essentially rewrite the the manifest URL _we wanted_ to be located on disk where the console _actually_ loaded from. e.g. the `/beta/preview_green/` part of the path was removed and would be located at `http://epix.xbox.com/epix/en-US/homepage.xml` instead and `epix.xbox.com` would resolve to `127.0.0.1`.

It's worth noting that these files were RSA signed and couldn't be tampered with:

[![Dashboard manifest header](/img/xbox-dashboard/epix-preview-signature.png)](/img/xbox-dashboard/epix-preview-signature.png)

Ditto with the XUI Lua scripts referenced by them (e.g. `http://epix.xbox.com/shaXam/0204/79/35/7935844a-91a8-45fe-a9c0-94dfa4d6c053.lzp?v=11#Beta_Announcements.lzp`)

This worked surprisingly well and for a long time we were happily accessing internal Xbox employee dashboards. With a C# utility to automate the work we were mirroring the manifest files every week or so and seeing what was new:

[![Screenshot of the mirrored dashboard channels](/img/xbox-dashboard/epix-mirrored.jpg)](/img/xbox-dashboard/epix-mirrored.jpg)

_Please ignore Ron Jeremy. I had no idea who he was at the time and had searched for "fat greasy man" with the intention of replacing all images in for the Canadian region (`en-CA`) with that photo to prank one of our friends in Canada who was using our server. It's unfortunate that this is one of the only clear screenshots of our MITM tricks that survived time._

### Manifests from Other Live Environments

A quick note about the dashboard manifest files: although the Xbox 360 had different signing keys for dev vs retail _executables and game content_, the dashboard files were signed with a shared key.

Through leaked internal Xbox 360 dev kit recoveries we were able to access alternate Xbox LIVE environments such as "int2", "vint", and a few others:

[![Xbox LIVE environments as seen on the Xbox 360 dev launcher](/img/xbox-dashboard/xbox-live-environments.jpg)](/img/xbox-dashboard/xbox-live-environments.jpg)


[![A folder of different Xbox LIVE environment files](/img/xbox-dashboard/live-environments-folder.png)](/img/xbox-dashboard/live-environments-folder.png)

Emma connected to int2 one day and discovered that the manifest files were set up for testing all of the available Xbox LIVE Gold offers... including Xbox LIVE for $1.

Since we could use these files on our retail Xboxes, we were able to continuously alternate between the Xbox LIVE Gold for free and Xbox Live Gold for $1 offers to get 1 year of Gold for $6/year.

### What's a "Preview Tool"?

Eventually on one of the dashboard channels I saw something weird that caught my eye:

[![A dashboard channel showing a Native American on a dashboard tile with the text "Xbox LIVE Marketplace Tools"](/img/xbox-dashboard/preview-tool-listing.jpg)](/img/xbox-dashboard/preview-tool-listing.jpg)

It's really hard to see on this terrible camera phone/CRT photo but the tile says "Xbox LIVE Programming Tools" at the very top. Upon navigating to the card there was an option to download something called "Preview Tool".

This is a little more clear to see on the app's boxart:

[![A dashboard channel showing a Native American on a dashboard tile with the text "Xbox LIVE Marketplace Tools"](/img/xbox-dashboard/preview-tool-modern-card.png)](/img/xbox-dashboard/preview-tool-modern-card.png)

The file was _very_ small and only displayed a message box asking if you'd like to enable "preview mode":

[![An Xbox message box with options to enable/disable preview mode](/img/xbox-dashboard/preview-tool-message-box.png)](/img/xbox-dashboard/preview-tool-message-box.png)

Upon enabling preview mode we suddenly saw debug information on the dashboard:

[![The Xbox dashboard with debug text printed out in the corner](/img/xbox-dashboard/preview-tool-debug.png)](/img/xbox-dashboard/preview-tool-debug.png)

Again, the text is _very_ hard to see but there is now red text in the top-right corner of the dashboard showing debug output including the frames per second the dashboard is rendering at. We also noticed that Preview Tool _on its own_ could force our console to use the internal dashboard files without having to do the MITM which made life much easier for us.

## Arbitrary Code Execution

Preview Tool was a unique type of application in that it actually had an expiration date associated with it. You were required to be on Xbox LIVE to launch the app and its revocation/expiration status would be checked by the system.

Sooner or later our copies of Preview Tool expired. Although we had the means of downloading anything we wanted from the Xbox LIVE marketplace we were too lazy to brute-force the randomized 32-bit ID required to download the newer packages that expired in the future. We had done this for other titles in the past but it was a process that took a couple days when distributed across multiple parties.

[![Brute forcing the Halo 4 beta offer ID with Archangel](/img/xbox-dashboard/halo4-offer-brute-force.jpg)](/img/xbox-dashboard/halo4-offer-brute-force.jpg)

_Archangel was a utility developed by [@xenomga9](https://twitter.com/xenomega9) for brute forcing the Halo 4 beta's offer ID. If you were ever wondering how the "[Halo 4 barn video](https://www.youtube.com/watch?v=POconAHU3aE)" came to be, this was it._

Brute forcing just wasn't worth the time and effort though when we were still doing CDN cloning for our friends without Preview Tool. So we just stopped getting Preview Tool and started doing MITM again.

We didn't realize how powerful Preview Tool was until our copies had expired. At some point I wanted to know how Preview Tool had been forcing the console to use internal dashboard channels, so I opened it up in IDA and noticed a call to an API I'd never seen before named `XamSetStagingMode`.

[![XamSetStagingMode function call](/img/xbox-dashboard/xam-set-staging-mode-call.png)](/img/xbox-dashboard/xam-set-staging-mode-call.png)

And the `XamSetStagingMode` API just sets some global:

[![XamSetStagingMode function disassembly](/img/xbox-dashboard/xam-set-staging-mode.png)](/img/xbox-dashboard/xam-set-staging-mode.png)

Ok... so who uses this global?

[![System staging mode global references](/img/xbox-dashboard/staging-mode-references.png)](/img/xbox-dashboard/staging-mode-references.png)

Interesting! `XamVerifyXSignerSignature` is used for verifying certain things which aren't checked by the hypervisor, like the dashboard manifest files and their Lua scripts. Checking how this is used:

[![XamVerifyXSignerSignature diassembly](/img/xbox-dashboard/staging-mode-check.png)](/img/xbox-dashboard/staging-mode-check.png)

Not shown in the above screenshot, but if staging mode is enabled and the file signature isn't valid the console will debug print the following string:

```
XamVerifyXSignerSignature: Signature not trusted, but ok since we're in staging mode or on a devkit
```

i.e. `XamSetStagingMode()`, and therefore Preview Tool, disables all signature checks of dashboard contents. *Correction: `XamVerifyXSignerSignature` will allow unsigned content if the caller provides a certain flag indicating they want to allow untrusted signatures in devkit/staging mode. Back when we originally discovered Preview Tool, the dashboard provided this flag but does not appear to anymore.*

This doesn't actually answer the question though of how the alternate dashboard channels were being used.

Although I never looked into it fully, I believe the dashboard or another component called `XamGetStagingMode()` and used a different value for the epix CDN/path. There also exists a "Live Hive" which is a key-value store used for dynamically configuring Xbox LIVE settings:

```
CatalogCDNUriPort=80
CatalogCDNUriRoot=http://catalog.vint.xboxlive.com
CatalogUriPort=80
CatalogUriRoot=http://catalog.vint.xboxlive.com
CloudStorageStatus=1
CommunityGamesTrialExpirationInSeconds=480
ContractManagerUriRoot=http://contractfd.test.xboxlive.com/v2
```

I'm almost positive one of these settings was overwritten by Preview Tool (or something else) to point at a different base URL for epix.

Although this is a bit anticlimactic, we barely even bothered to use this newfound knowledge of disabling signature checks since we had dev kits and could run our own _native_ code anyways. This only gave us arbitrary Lua scripting capabilities.

It also meant we'd have to grab Preview Tool versions that weren't expired every now and then, which we were too lazy to do. But it was nice to know that if we wanted to, we now knew how to control scripts on the dashboard.

In the end what we got from this entire effort was access to Xbox employee-only game betas, tools, and interesting insight into how the dashboard worked.

You can download some of the interesting manifest files I had saved on an old HDD [on Archive.org](https://archive.org/details/epix-playground-manifests). Unfortunately I did not mirror the Lua scripts and `epix-preview.xbox.com` has since been killed off by Microsoft -- but that's a story for another day :)
