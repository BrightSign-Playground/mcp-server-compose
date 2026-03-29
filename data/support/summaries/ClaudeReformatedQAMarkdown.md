### Question: How do I reprovision my device from a Partner Network to a bsn.content network?

Step 1. Please review the documentation here: https://docs.brightsign.biz/userguides/bsncontent.

Step 2. To use bsn.Content, please create a bsn.cloud network

Step 3. Purchase the bsn.Content subscriptions for the number of devices added. Contact our sales team at sales@brightsign.biz to get new subscriptions added to your network.

Step 4. Once the subscriptions are added, create a setup file for bsn.Content setup.

Step 5. In admin, go to the provision tab and add the bsn.Content setup you just created to the devices.

Step 6. Disconnect the player from power.

Step 7. Remove the SD card.

Step 8. Delete all files and folders from the card or format the card in exFAT.

Step 9. Factory reset the device; you can do this through: [Factory Reset a Player](https://docs.brightsign.biz/how-tos/factory-reset-a-player)

Step 10. Put the blank SD card back in the device

Step 11. Power up the device and connect it to internet, the device should setup.

### Question: How do I setup my new out of the box device for Local File networking?

Step 1. If you have not created a bsn.control network, please create a new network

Step 2. Create a new setup on the network for Local File Networking

Step 3. Go to the provision page in the Admin tab, and add the serial number(s) and assign it with the new setup file

Step 4. Power on the device with a blank SD card and connected to internet

Step 5. The player should download the setup the file and connect to the Network tab

Here is also a video for how to setup for Local file networking: https://www.youtube.com/watch?v=0GQ9RiurRNw&t=5s

### Question: My device is getting an error during provisioning, what should I check?

Step 1. For your device, we require certain ports and URL's to be able to access for can be found here:https://docs.brightsign.biz/advanced/bsncloud-ports-and-urls

Step 2. If you have continued issues, we would recommend checking that the setup file has the correct values for your networking options

Step 3. If you are using bsn.cloud setup, please make sure that you have active subscriptions for the device.

### Question: How do I put a new Setup file on the device?

Step 1. Create the new setup file.

Step 2. On the Provisioning tab, link the setup file to your player's serial number.

Step 3. Go to the Network tab and click on the remote Diagnostic Web Server (rDWS) for the device

Step 4. Go to the 'Control' tab in the rDWS and click on "Reprovision"

### Question: How do I reprovision my device from the Local Diagnostic Web Server?

Step 1. Login to the device from the ip address

Step 2. Insert username and password for the device

Step 3. Go to the SD tab, and delete the autorun.brs on the device

Step 4. Go to the control tab and select Factory reset

Step 5. The device should download the new setup package

### Question: Do you have a support call option?

We offer scheduled callbacks as a paid support service while we rely on our integrators and resellers for on site support.

Please have a look at the article for the standard and priority support callback options if that's something you would be interested in: https://docs.brightsign.biz/support

If purchasing a callback, please be sure to include

Step 1. Your existing support ticket number or a brief description of the issue.

Step 2. Your time zone.

Step 3. Specific dates and times you are available for at least the next three business days.

We will schedule a call in a new Zendesk ticket for the callback, so watch your email for mail from brightsign.zendesk.com

You can also utilize our tutorials, online learning, and tailored training options: https://www.brightsign.biz/resources/

### Question: What different options can I setup for my player?

You can setup the device as a bsn.Content, Local File networking, Standalone, Web Folder, or with a Partner Application.

### Question: When publishing with LFN, my device does shows unable to connect. What should I do?

Step 1. Check that port 8080 and port 8008 are enabled on your network and allows for HTTP transfers

Step 1a. You can see if a response is received from http://playerIPaddress:8080/GetID in a web browser.

Step 1b. You should also confirm you are on the same subnet for the device

Step 2. If the network is open, create a new LFN setup for the device and reprovision

### Question: I am getting a 401 or 403 error when trying to provision for bsn.cloud.

Step 1. Check that the ports are url's for bsn.cloud and provisioning are allowed on the network. The documentation can be found here: https://docs.brightsign.biz/advanced/bsncloud-ports-and-urls

Step 2. Check that you have active subscriptions for bsn.Content so that the device is able to use bsn.Content

Step 3. If you need to get active subscriptions, please reach out to orders@brightsign.biz to purchase. Provide your network, number of subscriptions, and administrator email for the network

### Question: When I try and provision my device, the provisioning is showing that the player already exists. Can you help me?

We do not have the ability to remove players from networks or change and view/reset passwords for BrightSign Network users for security and privacy reasons. The only way is for the previous admin to follow the linked steps below:

Move player
https://docs.brightsign.biz/how-tos/move-players-to-a-new-network

What if my player's serial number is already registered?
https://docs.brightsign.biz/faqs/troubleshooting#K2eWt

Step 1. If the previous administrator was with your company, we encourage you to reach out to your IT team to gain access through their account.

Step 2. If you are unable to contact the admin the only way to use the player is in stand alone or local mode in which you can still use the player but not in cloud mode.

Step 3. If the player was purchased second hand you will need to contact the seller to remove the player from their network

### Question: How do I get help with Partner content issues?

Step 1. If a customer is utilizing a Partner CMS not developed by BrightSign, the customer should reach out to the specific Partner CMS' support line for any software and content related issues.

Step 2. If a customer is running into hardware failures while running a partner CMS, they can continue to converse with the BrightSign support team at support@brightsign.biz.

Step 3. If provided the partner CMS in use and the customer has determined they are having a non-hardware issue, we can send them the corresponding support page and email for that Partner CMS:

A list of BrightSign CMS Partners and their support resources:

a. Appspace: They recommend going through the Appspace Account Portal with your Appspace email to file a ticket with their support team: https://community.appspace.com/p/support

b. Poppulo: For assistance with Poppulo software: https://www.poppulo.com/support . For arranging a call with the Poppulo team reach out through their 'Contact Us' form: https://www.poppulo.com/contact-us

### Question: How do I include Wi-Fi on a player that already has an ethernet connection

Step 1. Ensure you have the proper wireless module installed on your player. Series 5 and Series 6 players utilize the WD-105 Wireless module. Series 3 and 4 players use the WD-104 wireless module.

Step 2. In BSN.cloud create a Provisioning Record for the player using one of the following methods if one does not exist already:

Step 2a. [Add Player](https://docs.brightsign.biz/user-guides/provision#B5ZgG) from the Admin>Provision tab using its Serial Number.

Step 2b. [Activate](https://docs.brightsign.biz/user-guides/activate) the player Admin>Provision tab using the player's activation code.

Step 3. In BSN.cloud create a new Setup for your player, enabling Wi-Fi, and entering your wireless network's SSID and Passphrase. See [here](https://docs.brightsign.biz/user-guides/network-options#VeLDQ) for how to further alter wireless settings. Save the Setup to your [Setup Library](https://docs.brightsign.biz/user-guides/setup-library).

Step 4. Apply the previously created Setup to the player in the Admin>Provision tab. See [Apply Setup](https://docs.brightsign.biz/user-guides/provision#HGAN8) for more details.

Step 5. If the player is already in your BSN.cloud network, reprovision the player from the RDWS. [How to Reprovision a player](https://docs.brightsign.biz/user-guides/provision#nTaPt).

Step 6. If the player is not setup in your BSN.cloud network, insert a blank SD card and power on the player. The player will download and install the new wireless setup assigned to it in BSN.cloud.

### Question: How do I include Wi-Fi on a player that has no ethernet connection?

Step 1. Ensure you have the proper wireless module installed on your player. Series 5 and Series 6 players utilize the WD-105 Wireless module. Series 3 and 4 players use the WD-104 wireless module.

Step 2. In BSN.cloud create a Provisioning Record for the player using one of the following methods if one does not exist already:

Step 2a. [Add Player](https://docs.brightsign.biz/user-guides/provision#B5ZgG) from the Admin>Provision tab using its Serial Number.

Step 2b. [Activate](https://docs.brightsign.biz/user-guides/activate) the player Admin>Provision tab using the player's activation code.

Step 3. In BSN.Cloud create a new Setup for your player, enabling Wi-Fi, and entering your wireless network's SSID and Passphrase. See [here](https://docs.brightsign.biz/user-guides/network-options#VeLDQ) for how to further alter wireless settings.

Step 4. Save the Setup to locally to a blank micro-SD card on your computer.

Step 5. Apply the previously created Setup to the player in the Admin>Provision tab. See [Apply Setup](https://docs.brightsign.biz/user-guides/provision#HGAN8) for more details.

Step 6. Insert the micro-SD card with your new wireless setup files into your player and power on the device.

### Question: How do I install a Wireless module on my player?

Step 1. Ensure you have the correct wireless module for your player model. Series 5 and Series 6 players utilize the WD-105 Wi-Fi module. Series 3 and Series 4 players use the WD-104 Wi-Fi module.

Step 2. Refer to our documentation site for our respective installation guides and specs.

Step 2a. [WD-105 Installation Guides & Specs](https://docs.brightsign.biz/hardware/wd105-wi-fi-kit-series-5-and-6)

Step 2b. [WD-104 Installation Guides & Specs](https://docs.brightsign.biz/hardware/wd104-wi-fi-kit-series-3-and-4-bs-built-in)

### Question: How do I change Wi-Fi SSID and password on a unit that has already been provisioned?

Step 1. Update the Setup associated with the player's Provision Record to include the Wi-Fi settings. The existing setup can be modified within the Admin>Setup Library tab. See [here](https://docs.brightsign.biz/user-guides/network-options#VeLDQ)s for more details about specific Wi-Fi features.

Step 2. [Reprovision](https://docs.brightsign.biz/user-guides/provision#1lN4D) the player from the Control tab of the Remote Diagnostic Web Server.

### Question: I've forgotten my password/can't log-in to my BSN.cloud account. How do I reset my password?

For customers looking to reset their password, they can follow these steps assuming SSO is not being enforced by their email organization:

Step 1: Navigate to the [BSN.Cloud login page](https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/auth?client_id=baconnected&redirect_uri=https%3A%2F%2Fapp.bsn.cloud%2F%23%2Fadmin%2Factivate&state=c64d4487-c59e-4b26-9116-e306fa9e6fe4&response_mode=query&response_type=code&scope=openid&nonce=f652dbc3-3c02-4378-874f-b437839c7040&code_challenge=jTxCPS31mEke7cyOXuBQxQ3tQUbTA6kwzHgz20q_Q9E&code_challenge_method=S256).

Step 2: Select 'Try Another Way'.

Step 3: Select the 'Forgot my Password' option.

Step 4: Provide the email address associated with your BSN.Cloud account.

Step 5: Check your email for a password reset link and follow those instructions.

### Question: The person whose account the players are on no longer work at my company. How can I get access to the players?

Step 1: If the previous administrator was with your company, we encourage you to reach out to your IT team to gain access through their account.

### Question: How do I create a simple presentation?

Step 1: Please review our simple presentation video tutorial on our documentation website: [Video Tutorial](https://docs.brightsign.biz/how-tos/presentations#wWKXa)

### Question: How do I build a Multizone presentation?

Step 1: Please review our multizone presentation video tutorial on our documentation website: [Video Tutorial](https://docs.brightsign.biz/how-tos/presentations#_IYW_)

### Question: How do I Schedule or Publish a presentation/content?

Step 1: Please review our scheduling and publishing video tutorial on our documentation website: [Video Tutorial](https://docs.brightsign.biz/how-tos/presentations#0WPKs)

### Question: How do I setup a Partner CMS on my player?

Step 1: Please review our video tutorial for configuring a player to use a Partner CMS solution on our documentation website: [Video Tutorial](https://docs.brightsign.biz/how-tos/partner-cms#S2NPq)

### Question: What is BrightSign's End-of-life Policy for Series 2 players and older players?

Step 1: For Series 2 and older players, BrightSign no longer offers technical support. bsn.Content, bsn.Control, and any other BSN.cloud services do not support Series 2 and older players.

Step 2: Please refer to our documentation for the complete list of BrightSign products that have reached the end of technical support: [End of Life Matrix](https://docs.brightsign.biz/support/end-of-life-eol-policy#lOPmv)

### Question: What is BrightSign's End-of-life Policy for BSN (BrightSign Network) or BSNEE (BrightSign Network Enterprise Edition)?

Step 1: Software updates for BSNEE will stop in December of 2025 and will reach the end of support in December 2026.

Step 2: BSN & brightAuthor classic will shut down in March 2027. The final subscription renewals will end by March 2026.

Step 3: Players released after the EOL announcement on 7/30/2025 will be incompatible with BSN and BSNEE.

Step 4: For further details on our EOL policies, please refer to our website: https://www.brightsign.biz/end-of-life/

For more information surrounding player migration and their existing players, please have customers reach out to our sales team: sales@brightsign.biz.

### Question: What is BrightSign's End-of-life Policy for Series 3 players?

Step 1: Series 3 BrightSign players will reach the end of software support in April 2026. The last OS branch that will support Series 3 players is the 9.1.x branch.

Step 2: For the complete Series 3 EOL schedule please refer to our documentation.

Step 2a. Link - [Series 3 EOL Schedule](https://docs.brightsign.biz/support/series-3-end-of-life)

Step 2b. Link - [EOL Matrix for all BrightSign Products](https://docs.brightsign.biz/support/end-of-life-eol-policy#lOPmv)

### Question: What is Player Activation?

Step 1. Player Activation is an optional and alternate method of provisioning your player. If a provisioning record is already assigned in bsn.cloud, activation is not necessary, and the player will automatically load the assigned setup record.

Step 2. A player powered on with a blank storage device, an internet connection, and no existing provisioning record will then display the 'Player Activation' page.

Step 3. The page will display two methods for activation: A QR code and an Activation Code.

Step 4. The QR will allow users to Activate the player through their mobile device.

Step 5. The Activation code can be entered in the 'Admin>Activation' tab of a bsn.cloud network in BrightAuthor:Connected or bsn.cloud.

Step 6. Once a player has been activated on a bsn.cloud network, the player will download the 'Default Setup' specified on their network. By default, this is set to LFN (Local File Networking).

### Question: Is Player Activation required for player setup / player provisioning?

Step 1. No, player activation is an optional and alternate method for provisioning your player.

Step 2. If your player is already provisioned and online in your BSN.cloud network by creating and assigning a setup record in bsn.cloud, then player activation is not necessary.

### Question: How do I move a bsn.Content subscription from one network to another:

- Please refer to our documentation page with detailed step-by-step instructions. Link: [Move Players to a New Network](https://docs.brightsign.biz/how-tos/move-players-to-a-new-network)

### Question: I don't see my local content when logging into BSN.cloud on a different computer.

When creating a bsn.control presentation on Computer A, the presentation and files will not be present when you log into BSN.cloud on Computer B.

To alter an existing bsn.control presentation on a different computer, you will need a locally saved .bpfx file of the presentation and copies of all the media used in the presentation.

### Question: How do I upload media to BSN.content network?

Step 1: Select the 'Content' tab within BA:Connected.

Step 2: Select the 'Upload New Media' field.

Step 3: Upload your desired media content.

For additional information on uploading media content, please refer to our documentation: [Uploading media to BSN.content](https://docs.brightsign.biz/user-guides/managing-content)

### Question: My player shows under the network tab but is inactive.

If your player is showing as inactive in the network, it means that it has not successfully checked in with BSN.Cloud for over 36 minutes. Here are some steps you can take to troubleshoot the issue:

Step 1 (Check Power and Connections): Ensure that the player is powered on and physically connected to the network. Verify that the Ethernet cable is plugged in and that the Ethernet LEDs are blinking as expected.

Step 2 (Check Network Connectivity): Confirm the player is ethernet or WiFi connection is online and has access to the required ports and URLs for BSN.cloud: [BSN.cloud Ports and URLs](https://docs.brightsign.biz/advanced/bsncloud-ports-and-urls)

Step 3 (Check for Configuration Issues): Ensure that the player is configured correctly to connect to BSN.Cloud. This includes checking that the correct setup has been applied and that the player is enabled for BSN.Cloud.

Step 4 (Confirm there is no BSN.cloud maintenance period): Check the BSN.cloud network status in BA:Connected.

If the player remains inactive after these steps, you can reach out to BrightSign support for further assistance over email - support@brightsign.biz.

### Question: I don't see my player in the listed 'Networked Players' when attempting to publish a presentation.

Step 1: Ensure the player is connected via its IP address under the 'Network Players' section.

Step 2: If the player is not displayed, select the '+' icon to add the player to its IP address.

If the player is still not displaying, you can reach out to BrightSign support for further assistance over email - support@brightsign.biz.

### Question: How can I schedule multiple presentations in the same schedule?

Step 1. Drag and drop desired presentations from the 'Recent Presentations' tab to the schedule.

Step 2. Select the presentation to alter its scheduled time to play.

Step 3. Save and publish the schedule to your player.

For specific details and implementations please refer to our docs page here: [Scheduling and Publishing - User Guides](https://docs.brightsign.biz/user-guides/scheduling-and-publishing)

### Question: How do I publish the same presentation to different player models?

You can publish the same presentation to different player models but be cognizant that different models contain different feature sets.

We recommend validating content types and presentation configurations on a per player model basis.

### Question: Why is there a lock icon on my BSN.content presentation?

The lock icon indicates your presentation is currently scheduled to a player.

### Question: Why does my content have a lock icon on it?

A lock icon on your content indicates that it is currently used in an active presentation. When locked, the content cannot be deleted or modified. To make changes to the content, it must first be removed from the active presentation.

### Question: How do I ensure content is supported for my player model?

Step 1 (Check Supported Resolutions): Refer to the documentation for your specific player model to find the maximum resolution and codecs it can handle: [Video Formats and Codecs](https://docs.brightsign.biz/advanced/video-formats-and-codecs)

Step 2 (Re-encode content): If your media content is not a supported resolution or format, you can use third party software to re-encode and re-size to your players supported specifications.

### Question: What is the Live Feed widget?

The Live Feed widget can be used in presentations to display media from an MRSS feed or text from an RSS feed in Ticker zones.

Please refer to our Live Feed Widget documentation: [Live Feed Widget](https://docs.brightsign.biz/user-guides/live-feed-state)

### Question: What is the stream widget?

The Stream widget can be used in BrightSign presentations to play various types of streams, including audio, video, and MJPEG streams.

Please refer to our Stream Widget documentation: [Stream Widget](https://docs.brightsign.biz/user-guides/stream-state)

### Question: What is the HTML widget?

The HTML widget can be used in BrightSign presentations to display HTML web pages, including video, images, text, and JavaScript elements.

Please refer to our Documentation: [HTML 5 Widget](https://docs.brightsign.biz/user-guides/html5-state)

### Question: Do BrightSign Players support PowerPoint (PPT) files?

ASK tier 1 what their step by step anser to this would be. THROW into support general

BrightSign players do not support playback of native PowerPoint (PPT) files.

It is possible to export your PowerPoint presentation to a series of .BMP or .JPEG images and play those images using a BrightAuthor presentation. It is also possible to export your PowerPoint into an MPEG4 .mp4 file to be played by the BrightSign player.

### Question: How do I play a YouTube Video?

Please refer to our BrightSign GitHub repository with detailed information on displaying YouTube Videos: [Link to BrightSign YouTube Guide](https://github.com/srodriguez-brightsign/bs-youtube-iframe)

### Question: How do I install an SSD on my BrightSign Player?

Please refer here to find the SSD installation guide for your corresponding player model: [WiFi, SSD, and Cell Modem Info](https://docs.brightsign.biz/hardware/wi-fi-ssd-and-cell-modem-info)

### Question: What kind of SSD can I use on the BrightSign Player?

BrightSign players all feature an M-keyed M.2 connector for connecting SSD drives. 

For Series 4, 5, and 6 players we support NVME SSD types for both 2242 and 2280 sizes. 

For recommendations on specific minimum read/write speeds or SSD info for Series 3 players please refer to our complete documentation here: [SSD Requirements](https://docs.brightsign.biz/hardware/ssd-requirements)

### Question: How can I communicate with my TV from a BrightSign player?

Step 1. RS232: You can connect an RS232 cable to extended I/O devices that will allow you to pass information to the device.  

Step 2. CEC/BrightControl: You can send commands through CEC if the TV supports the commands. Specific CEC commands, such as turn on/off for generic displays are possible through the BrightControl option in the advanced commands for BA:connected. More information can be found here: [BrightControl User Guide](https://docs.brightsign.biz/user-guides/brightcontrol)

Step 3. If you are not using BA:Connected for publishing, we recommend reaching out to your CMS provider to get their best recommendations.

### Question: How do I turn on or off my television with my player?

If you are using BA:Connected, then you can follow instructions here to enable power save mode on the display using CEC Commands: [How to Sleep/Wake a display](https://docs.brightsign.biz/how-tos/sleep-wake-a-display)

### Question: How do I send or receive UDP messages to my player?

Please refer to our documentation for how to receive UDP messages within a BA:Connected presentation: [Use Serial / UDP Commands](https://docs.brightsign.biz/how-tos/use-serial-udp-commands) & [Receiving UDP Message + Trigger Example](https://docs.brightsign.biz/how-tos/send-a-serial-command-on-receiving-a-udp-message)

### Question: How can I setup a proxy on my player?

A proxy can be configured within a player setup record in bsn.cloud.  

When creating your setup record, select ‘Network Options > Player > Host Configuration > Use Proxy’ to enter your proxy information. Please refer to our documentation for specifics on formatting the proxy address: [Proxy Setup Record](https://docs.brightsign.biz/user-guides/network-options#host-configuration)

### Question: How do I create a bsn.cloud account?

To create a new bsn.cloud account you have two options: 

Option 1: You can sign up directly on our website at [https://bsn.cloud/](https://bsn.cloud/) and select the ‘Sign Up’ icon at the top of the page.

Option 2: The second option is to create your account while logging into our desktop application BA:Connected. - [Create Account in BA:Connected Guide](https://docs.brightsign.biz/how-tos/create-a-bsncloud-account-and-network)

    Step 1. Launch BA:Connected
    Step 2. Select 'Connect to BSN.Cloud'
    Step 3. Select the 'New User?' option at the bottom of the sign-in window and fill out the prompts.

### Question: How can I add my content and publish it to my player on bsn.Control cloud?

Step 1. Log into your bsn.Control network through the brightAuthor:Connected desktop application. 

Step 2. In the drop down of the ‘Presentation’ tab either create a new presentation, open an existing .bpfx presentation from your local computer, or import a presentation using a local .bpfx file with the access to the necessary content files. 

Step 3. Once in your presentation editor, identify the zone you would like to assign content to and double click the zone. 

Step 4. Find the ‘Media’ tab on the left side of the screen and select the 'Open’ button. From here you can open a local folder on your computer that contains your image and media files. 

Step 5. Select the ‘Assets’ tab at the bottom of the presentation editor. The content stored in your selected local folder should now be accessible to drag into your Zone in the center of the screen.  

Step 6. After adding the desired content to your presentation, select the ‘save’ icon at the top of the editor and then ‘Publish’.

For additional information on the different publishing types on bsn.Control (Standalone, Local File Networking, and Web Folder), please refer to our documentation here: [Standalone Player Publish](https://docs.brightsign.biz/publishing-quickstart#publish-to-a-standalone-player)

For information on scheduling presentations please refer to our documentation here: [Scheduling](https://docs.brightsign.biz/user-guides/schedule)

### Question: How can I add my content and publish it to my player on bsn.Content cloud?

### Uploading Content into your Media Library

Step 1. Log into your bsn.Content network through bsn.cloud or the brightAuthor:Connected desktop application. 

Step 2. Select the drop-down arrow on the ‘Content’ tab and select ‘Upload New Media’ 

Step 3. From this page you can select the images and videos you would like to upload into your bsn.Content ‘Media Library’.  These cloud assets can be used in any newly created bsn.Content presentation.

### Adding Cloud content to your Presentation

Step 1. In the drop down of the ‘Presentation’ tab in BSN.cloud, either create a new presentation, open an existing presentation from your presentation library, or import a presentation using a local .bpfx file with the access to the necessary content files. 

Step 2. Once in your presentation editor, identify the zone you would like to assign content to and double click the zone. 

Step 3. Find the ‘Media’ tab on the left side of the screen and select the 'Media Library’ folder you uploaded your desired content to. 

Step 4. Select the ‘Assets’ tab at the bottom of the screen. The content stored in your selected ‘Media Library’ folder should now be accessible to drag into your Zone in the center of the screen.  

Step 5. After adding the desired content to your presentation, select the ‘save’ icon at the top of the editor and then ‘Publish’. 

Step 6. Once on the publishing/scheduling page, on the left side of the screen under ‘Destination’, select the ‘Group’ dropdown to select which bsn.Content group you would like your presentation published to.

For additional information on customizing your bsn.Content publishes, please refer to our documentation here: [Publish to bsn.Content](https://docs.brightsign.biz/publishing-quickstart#publish-to-bsncontent)

For information on scheduling your presentations please refer to our documentation here: [Scheduling](https://docs.brightsign.biz/user-guides/schedule)