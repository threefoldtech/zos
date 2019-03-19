# Requirements 
- A component can register one or more `objects` 
- Each object can have a global name, where clients can use to call `methods` on the registered objects
- Objects interface need to explicitly set a version. Multiple versions of the same interface can be running at the same time
- A client, need to specify the object name, and interface version. A Proxy object can be used to call the remote methods
- Support mocked proxies to allow local unit tests without the need for the remove object

## Suggested message brokers
- Redis
- Disque
- Rabbitmq

> I think disque is a really nice option

## Overview
