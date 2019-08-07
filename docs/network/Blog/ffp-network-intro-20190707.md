## Zero-OSv2 and networking

### Intro
We've been quite busy lately on that new thingie called Zer-OS. This time, like the first time, we started from scratch, having learned a lot from the first time.
The King is dead, long live the King? Maybe not. Zero-OS is no king, it's just an abstraction layer to hardware for running tings that users can see/interact with.  
But these abstraction layers need to be there, and OS developers are the guys in the top of the heap. We're just piggybacking on already existing wonderfull things these guys wrote, and we're thankful for it.

So ... Zero-OS runs on LINUX. The kernel that is. And some small tools to get that thing going.  
But once that kernel is running and has discovered all hardware, Zero-OS is quite a different beast compared to your mom&pop Linux (or should we say GNU Linux?) distribution.

For instance, we never run services that are visible on the network in the host itself. Even more, we block all possible calls to the machine, almost akin to just having cut the ethernet cable to the machine. Talk about attack surface? Nah, not in this house.

Now that is all well and dandy, but you need networking, right? You need to be able to access the node to do things, right?  
For networking, well of course you do! But not like you're used to.  
And for accessing the node? Hell no you don't! You're a user, you want to expose things your users want to use, why would you want an OS sit in your way?  
For all the user stuff, we'll refer you to another Blog, let's talk about networking.

### Network
While you're used to just type in a webpage in the textbox on top of your browser, or (god forbid) google it, you know you can just clickety-click through te web to you heart's content. But under the hood... Man! you'd be amazed on how many things need to go exactly right to show you the page you're reading. Even for us, it's mind-boggling.

Let alone when you start to layer things on top of one another. And encrypt..., and keep it fast, and that it still looks like 'normal'.

But anyway.

When using the Grid (as a grid-user that is, i.e. someone who wants to use the grid and run services for HIS users to access/use), you need to buy a provision to run your service on that grid. That provision encompasses a place where you want to run that service, how big your requirements are in terms of memory/cpu/storage and... a network.  

So you get a network. A virtual one... like... not a real one, but behaves like a real one. That network runs on top of the real network (aka The Internet).  
Many solutions exist; some can hardly be called a solution, others are extremely difficult to setup, and some are 'a good idea', but are hindered by many constraints that suddenly make it less than optimal (eufemism), some are.. erm.. not cheap or even worse.. not open source.

So: You get a network. An overlay network that is. What does that mean?

Your network has it's own IP-range, it's own interfaces and once you're connected to that network, it just looks like any other. It has routing tables, has peers, you can communicate with your services, but it lays on top of the 'other' network, and all communications are encrypted so no one can snoop on your communications. A bit like a vpn, but for all services, and every service has a vpn.

The main building block is [Wireguard](https://wireguard.io), another thing very smart people created, but it's just that, a building block. To have every service communicate with every other service, with rules wich service can or can not reach another, and keep it simple enough so it doesn't get in the way, requires a lot of automation and fine-grained control.  
Right now, we have a 'Diamond in the rough' where the first functionality is: get it all connected and make sure every service in that (encrypted) network can communicate. Also make sure there is also an access point from 'The Outside' (Internet), to services on 'The Inside' and have basic access control on the edge of your Network to The Inside, so that at least you have a door with a key to your house.

That's where we are, next Blog will talk about how to set up that network yourself, adn where we go from there.

