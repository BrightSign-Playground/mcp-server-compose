# BrightSign Support Knowledge Base

**Source:** BrightSign Technical Support
**Category:** Setup, Provisioning, Networking, Hardware, Content, End-of-Life
**Audience:** BrightSign customers, integrators, and resellers

---

## Overview

This knowledge base covers the most common setup, provisioning, networking, hardware, and content questions for BrightSign players and the bsn.cloud ecosystem. Topics include provisioning players to bsn.Content and bsn.Control networks, configuring Wi-Fi and wireless modules, using the Local and Remote Diagnostic Web Servers, troubleshooting inactive players and LFN publishing failures, end-of-life timelines for BSN and BSNEE, and managing content in bsn.Content and bsn.Control networks.

BrightSign players can be configured for five operating modes: bsn.Content, bsn.Control (Local File Networking), Standalone, Web Folder, and Partner Application. The bsn.cloud platform (accessed via bsn.cloud or the BrightAuthor:Connected desktop application) is the primary management interface for all networked deployments.

---

## Network Types and Setup Options

BrightSign players support multiple deployment modes:

- **bsn.Content** — Cloud-managed content delivery. Requires a bsn.cloud network and active per-device bsn.Content subscriptions. Content is uploaded to a Media Library in the cloud and published to devices via groups.
- **bsn.Control (Local File Networking)** — Content is published from a local computer to players on the same network. Requires ports 8080 and 8008 to be open and allowing HTTP transfers.
- **Standalone** — Player runs content from local storage with no network management.
- **Web Folder** — Player pulls content from a web server.
- **Partner Application** — Player runs a third-party CMS application.

For Local File Networking, verify connectivity by opening `http://playerIPaddress:8080/GetID` in a browser. A response confirms the player is reachable. Confirm the publishing computer and player are on the same subnet.

---

## Creating a bsn.cloud Account and Network

To create a new bsn.cloud account:

**Option 1 — via the website:** Navigate to bsn.cloud and select the Sign Up icon at the top of the page.

**Option 2 — via BA:Connected:** Launch BrightAuthor:Connected, select Connect to BSN.Cloud, then select the New User? option at the bottom of the sign-in window and complete the prompts.

Once an account is created, a bsn.cloud network is automatically provisioned. For bsn.Content deployments, purchase per-device subscriptions by contacting orders@brightsign.biz and providing the network name, the number of subscriptions needed, and the administrator email address. For general sales inquiries, contact sales@brightsign.biz.

---

## Provisioning and Setup Files

### What is a Provisioning Record?

A Provisioning Record links a player's serial number to a setup file in bsn.cloud. It is created in the Admin > Provision tab either by adding the player's serial number directly or by activating the player using its Activation Code.

### What is a Setup File?

A Setup file defines how a player connects to a network — including the network type (bsn.Content, LFN, etc.), Wi-Fi credentials, proxy settings, and other network options. It is created in bsn.cloud and saved to the Setup Library. Setup files are assigned to players via the Provisioning Record.

### How to Provision a New Device for Local File Networking

1. Create a bsn.Control network in bsn.cloud if one does not exist.
2. Create a new setup for Local File Networking.
3. Go to Admin > Provision, add the player's serial number, and assign the new setup.
4. Power on the device with a blank SD card and an internet connection.
5. The player downloads the setup file and appears under the Network tab.

A video tutorial is available: https://www.youtube.com/watch?v=0GQ9RiurRNw&t=5s

### How to Put a New Setup File on an Already-Provisioned Device

1. Create the new setup file in bsn.cloud.
2. On the Provisioning tab, link the setup file to the player's serial number.
3. Go to the Network tab and open the Remote Diagnostic Web Server (rDWS) for the device.
4. In the rDWS, go to the Control tab and click Reprovision.

### How to Reprovision a Device from the Local Diagnostic Web Server (lDWS)

1. Open a browser and navigate to the player's IP address to access the lDWS.
2. Enter the username and password for the device.
3. Go to the SD tab and delete `autorun.brs` from the device.
4. Go to the Control tab and select Factory Reset.
5. The device downloads the new setup package and reprovisioning completes.

### How to Reprovision from a Partner Network to bsn.Content

1. Review the bsn.Content documentation at docs.brightsign.biz/userguides/bsncontent.
2. Create a bsn.cloud network.
3. Purchase bsn.Content subscriptions for the number of devices. Contact sales@brightsign.biz.
4. Once subscriptions are active, create a setup file configured for bsn.Content.
5. In Admin > Provision, apply the new bsn.Content setup to the target devices.
6. Disconnect the player from power and remove the SD card.
7. Delete all files from the card or format it as exFAT.
8. Factory reset the device. See: docs.brightsign.biz/how-tos/factory-reset-a-player
9. Reinsert the blank SD card.
10. Power on the device with an internet connection. The device provisions automatically.

---

## Player Activation

Player Activation is an optional, alternate method of provisioning a player. It is not required if a Provisioning Record has already been assigned to the player in bsn.cloud — in that case, the player automatically loads its assigned setup when powered on with a blank SD card and an internet connection.

When a player is powered on with a blank storage device, an internet connection, and no existing Provisioning Record, it displays the Player Activation page. This page presents two activation methods:

- **QR Code** — Scan with a mobile device to activate.
- **Activation Code** — Enter the code in the Admin > Activation tab of a bsn.cloud network in BrightAuthor:Connected or bsn.cloud.

Once activated, the player downloads the network's Default Setup, which is set to Local File Networking (LFN) by default.

---

## Wi-Fi and Wireless Modules

### Compatible Wireless Modules

- **WD-105** — Compatible with Series 5 and Series 6 players.
- **WD-104** — Compatible with Series 3 and Series 4 players.

Installation guides and hardware specifications:
- WD-105: docs.brightsign.biz/hardware/wd105-wi-fi-kit-series-5-and-6
- WD-104: docs.brightsign.biz/hardware/wd104-wi-fi-kit-series-3-and-4-bs-built-in

### Adding Wi-Fi to a Player with an Existing Ethernet Connection

1. Install the correct wireless module (WD-105 for Series 5/6, WD-104 for Series 3/4).
2. In bsn.cloud, create a Provisioning Record for the player if one does not exist, using Add Player via serial number or Activate via activation code.
3. Create a new Setup in bsn.cloud with Wi-Fi enabled, entering the SSID and passphrase. Save to the Setup Library.
4. Apply the Setup to the player in Admin > Provision.
5. If the player is already on the network, reprovision from the rDWS Control tab.
6. If the player is not yet on the network, insert a blank SD card and power on. The player downloads the new wireless setup.

### Adding Wi-Fi to a Player with No Ethernet Connection

Follow the same steps as above, except at step 4 save the Setup locally to a blank micro-SD card on your computer instead of applying it remotely. Insert the SD card into the player and power on.

### Changing Wi-Fi SSID and Password on an Already-Provisioned Player

1. Update the Setup associated with the player's Provisioning Record in Admin > Setup Library. Add or update the Wi-Fi SSID and passphrase.
2. Reprovision the player from the Control tab of the Remote Diagnostic Web Server.

---

## Proxy Configuration

A proxy is configured within the player's setup record in bsn.cloud. When creating or editing a setup record, navigate to Network Options > Player > Host Configuration > Use Proxy and enter the proxy address. Refer to docs.brightsign.biz/user-guides/network-options#host-configuration for proxy address formatting details.

---

## Troubleshooting

### Player Shows as Inactive in bsn.cloud

A player showing as inactive has not successfully checked in with bsn.cloud for over 36 minutes. Troubleshooting steps:

1. **Check power and connections** — Confirm the player is powered on and the Ethernet cable is connected. Verify Ethernet LEDs are blinking.
2. **Check network connectivity** — Confirm the player's Ethernet or Wi-Fi connection has access to the required BSN.cloud ports and URLs. See: docs.brightsign.biz/advanced/bsncloud-ports-and-urls
3. **Check configuration** — Verify the correct setup has been applied and the player is enabled for bsn.cloud.
4. **Check for maintenance** — Review the bsn.cloud network status in BA:Connected.

If the player remains inactive after these steps, contact support@brightsign.biz.

### LFN Publishing Shows "Unable to Connect"

1. Verify that ports 8080 and 8008 are open on the network and allow HTTP transfers.
2. Open `http://playerIPaddress:8080/GetID` in a browser to confirm a response is received.
3. Confirm the publishing computer and player are on the same subnet.
4. If the network is confirmed open and the issue persists, create a new LFN setup for the device and reprovision.

### Provisioning Error (401 or 403)

1. Verify that the required bsn.cloud ports and URLs are allowed on the network. See: docs.brightsign.biz/advanced/bsncloud-ports-and-urls
2. Confirm the device has active bsn.Content subscriptions. Contact orders@brightsign.biz to purchase subscriptions, providing your network name, required subscription count, and administrator email.

### Player Already Exists on Another Network

BrightSign cannot remove players from networks or reset account passwords for security reasons. Only the previous network administrator can release the player. Options:

- If the previous admin was at your company, contact your IT team to regain access.
- If you cannot reach the previous admin, the player can be used in Standalone or Local File Networking mode but not in cloud mode.
- If the player was purchased second-hand, contact the seller to have it removed from their network.

Reference: docs.brightsign.biz/how-tos/move-players-to-a-new-network

### Player Does Not Appear in Networked Players During Publishing

1. Ensure the player is connected via its IP address under the Network Players section.
2. If not displayed, select the + icon to add the player by IP address.

Contact support@brightsign.biz if the player still does not appear.

---

## Moving a bsn.Content Subscription

To move a bsn.Content subscription from one network to another, follow the step-by-step instructions at: docs.brightsign.biz/how-tos/move-players-to-a-new-network

---

## End-of-Life Policy

### BSN (BrightSign Network) and brightAuthor Classic

BSN and brightAuthor classic will shut down in **March 2027**. Final subscription renewals will end by **March 2026**. Players released after the EOL announcement on July 30, 2025 are incompatible with BSN and BSNEE.

### BSNEE (BrightSign Network Enterprise Edition)

Software updates for BSNEE stop in **December 2025**. BSNEE reaches end of support in **December 2026**.

For player migration assistance, contact sales@brightsign.biz. Full EOL details: brightsign.biz/end-of-life

### Series 3 Players

Series 3 players reach end of software support in **April 2026**. The last supported OS branch for Series 3 is the **9.1.x branch**. See the complete Series 3 EOL schedule at docs.brightsign.biz/support/series-3-end-of-life

### Series 2 and Older Players

BrightSign no longer offers technical support for Series 2 and older players. bsn.Content, bsn.Control, and all bsn.cloud services do not support Series 2 and older players. See the full EOL matrix at docs.brightsign.biz/support/end-of-life-eol-policy

---

## TV Communication

### RS232

Connect an RS232 cable to extended I/O devices to pass commands to the display.

### CEC via BrightControl

If the TV supports CEC, commands such as power on/off can be sent through the BrightControl option in Advanced Commands within BA:Connected. This includes generic display sleep/wake commands. See: docs.brightsign.biz/user-guides/brightcontrol and docs.brightsign.biz/how-tos/sleep-wake-a-display

If not using BA:Connected, contact the CMS provider for their recommendations on TV control.

### UDP Messages

Receive UDP messages within a BA:Connected presentation using the methods described at: docs.brightsign.biz/how-tos/use-serial-udp-commands

---

## Hardware

### SSD

All BrightSign players use an M-keyed M.2 connector for SSD drives. Series 4, 5, and 6 players support NVMe SSDs in both 2242 and 2280 sizes. For minimum read/write speed requirements and Series 3 SSD compatibility, see: docs.brightsign.biz/hardware/ssd-requirements

Installation guides by player model: docs.brightsign.biz/hardware/wi-fi-ssd-and-cell-modem-info

---

## Content Management

### Uploading Content to bsn.Content

1. Log into the bsn.Content network via bsn.cloud or BA:Connected.
2. Select the Content tab and choose Upload New Media.
3. Upload images, videos, and other media. Uploaded assets are stored in the Media Library and available for use in any bsn.Content presentation.

Reference: docs.brightsign.biz/user-guides/managing-content

### Publishing to bsn.Content

1. In the Presentation tab, create a new presentation or open an existing one.
2. Double-click a zone in the presentation editor to assign content.
3. In the Media tab, select the Media Library folder containing the uploaded content.
4. Drag content from the Assets tab into the zone.
5. Save the presentation and select Publish.
6. On the publishing page, under Destination, select the Group to publish to.

### Publishing to bsn.Control (Local)

1. Log into bsn.Control via BA:Connected.
2. In the Presentation tab, create or open a presentation.
3. Double-click a zone to assign content.
4. In the Media tab, open a local folder containing the media files.
5. Drag content from the Assets tab into the zone.
6. Save and Publish.

Note: bsn.Control presentations are stored locally on the computer where they were created. To edit on a different computer, bring a locally saved .bpfx file and copies of all media files used.

### Lock Icons

- A **lock icon on a presentation** indicates the presentation is currently scheduled to a player.
- A **lock icon on content** indicates the content is used in an active presentation and cannot be deleted or modified until it is removed from the presentation.

### Supported Content Types

BrightSign players do not support native PowerPoint (.ppt/.pptx) files. Workarounds: export slides to BMP or JPEG images, or export to MPEG4 (.mp4). For video codec and resolution support by player model, see: docs.brightsign.biz/advanced/video-formats-and-codecs

### YouTube

For displaying YouTube videos on BrightSign players, refer to the BrightSign GitHub guide: github.com/srodriguez-brightsign/bs-youtube-iframe

---

## Partner CMS Support

For issues with a third-party Partner CMS, contact the CMS vendor's support directly. BrightSign support handles hardware failures only when a partner CMS is in use.

If the CMS is identified, BrightSign support can provide the corresponding vendor support contact:

- **Appspace** — File a ticket via the Appspace Account Portal using your Appspace email: community.appspace.com/p/support
- **Poppulo** — Software support: poppulo.com/support — Call scheduling: poppulo.com/contact-us

For a video tutorial on configuring a player for a Partner CMS: docs.brightsign.biz/how-tos/partner-cms

---

## Support Options

BrightSign offers email support at support@brightsign.biz. Scheduled callback support is available as a paid service. To purchase a callback, provide:

1. Your existing support ticket number or a brief issue description.
2. Your time zone.
3. Available dates and times for at least the next three business days.

Callbacks are scheduled via a new Zendesk ticket — watch for email from brightsign.zendesk.com.

Additional resources including tutorials, online learning, and training: brightsign.biz/resources
