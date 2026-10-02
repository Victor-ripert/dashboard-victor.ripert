
# Frontend

---

## Project context

Our dashboard must display multiple configurable widgets. Users should be able
to add, configure, move and remove widget instances. Different instances of
the same widget may display different data, and each instance can have its own
refresh interval.

We considered three established options for building the frontend:

- React
- Vue
- Angular

These technologies are [widely used and have substantial documentation,
communities and tooling](https://gist.github.com/tkrotoff/b1caa4c3a185629299ec234d2314e190). We selected them as candidates rather than assuming
that popularity alone makes one of them the best fit for our project.

## Team experience and learning cost

Our team has some experience with React, but little or no practical experience
with Vue or Angular. It does reduce implementations risks.

Angular seams more complicated and hard to learn, but adopting Angular conventions would add a learning step for
our team and some time lose. Vue is also a credible alternative; we simply do not have the same
starting familiarity with it too.

So here React seams way better.

## Maintenance and long-term viability

A frontend technology must remain maintained throughout the project and have
documentation that helps us upgrade and fix problems. React, Vue and Angular
are all established options. We found no project-specific reason to reject Vue
or Angular on maintenance grounds.

Here both 3 options seams asame so it doesn't influencing the technologie choice. 

## Ecosystem and community

React is the most used frontend framework in the world, the ecosysteme is massive, Vue and Angular are huge too but the community on react is 2 time bigger thant both combined.

So here another point for react. At this point we can already take react as a frontend framework.

## Decision

We choose **React** for the frontend. Because of the previous analysis.

### Tailwind

Tailwind and bootstrap are comparable but here by preferences we choose tailwind to style our dashboard.

---

# Backend

---

We compared Go with Gin, Node.js with Express and Python with FastAPI for the
backend gateway of our dashboard.

The gateway receives requests from the frontend, authenticates users, validates
requests and forwards them to the appropriate services.

## Team experience

Our team has more experience with JavaScript and Python than with Go. Node.js
and FastAPI would therefore have a lower initial learning cost.

However, we want to use this project to learn Go, while keeping its role
limited to the gateway. Business logic can remain in dedicated services when
needed.

## Maintenance and tooling

All three options are actively maintained and have mature ecosystems.

Express is a lightweight Node.js framework that relies on middleware modules.
FastAPI provides automatic API documentation, type hints and middleware
support. Gin provides routing, JSON responses, route groups and middleware
support for concerns such as logging, error recovery, authentication and CORS.

## Fit for the project

Our dashboard uses several services. We need a lightweight gateway that can
handle many HTTP requests, centralize cross-cutting concerns and communicate
with multiple backend services **fast**.

Go is a compiled and statically typed language. It's very optimized for simple tasks and very good for scalability than python and JS.

Gin fits this role because its route groups and middleware make it easy to
organize public routes, authenticated routes and service proxy routes.

## Decision

We chose **Gin** for the API gateway.

The main reasons are its suitability for concurrent network services, its
simple deployment model and Gin's middleware-based architecture. This gives us
a clean central point for routing, authentication, CORS, logging and error
handling.
