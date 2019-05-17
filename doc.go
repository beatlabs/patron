/*
Package patron framework

Patron is a framework for creating microservices.

Patron is french for template or pattern, but it means also boss which we found out later (no pun intended).

The entry point of the framework is the Service.
The Service uses Components to handle the processing of sync and async requests.
The Service can setup as many Components it wants, even multiple HTTP components provided the port does not collide.
The Service starts by default a HTTP component which hosts the debug, health and metric endpoints.
Any other endpoints will be added to the default HTTP Component as Routes.
The service set's up by default logging with zerolog, tracing and metrics with jaeger and prometheus.

Patron provides abstractions for the following functionality of the framework:

  - service, which orchestrates everything
  - components and processors, which provide a abstraction of adding processing functionality to the service
  	- asynchronous message processing (RabbitMQ, Kafka)
  	- synchronous processing (HTTP)
  - metrics and tracing
  - logging
  - configuration management

Patron provides same defaults for making the usage as simple as possible.
For more details please check out github repository https://github.com/beatlabs/patron.
*/
package patron
