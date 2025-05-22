# PetStore Microservices Project (PetPal Connect)

## 1. Project Overview and Topic

PetPal Connect is a microservices-based application designed to simulate a modern Pet Store's backend operations. It allows users to manage their accounts, browse and list pets, apply for pet adoptions, and receive notifications about their applications. The system is built with a focus on clean architecture, scalability, and resilience, utilizing gRPC for inter-service communication and a message queue for asynchronous operations.

The primary goal is to provide a platform where:
* Users can register, login, and manage their profiles.
* Pets can be listed with details like species, breed, age, and adoption status.
* Users can apply for pet adoptions.
* Adoption applications can be processed, and users are notified of status changes via email.

## 2. Technologies Used

* **Programming Language:** Golang (Go)
* **API Gateway Framework:** Gin (for RESTful endpoints)
* **Inter-service Communication:** gRPC
* **Data Definition:** Protocol Buffers (Proto)
* **Databases:**
    * MongoDB (for persistent storage for User, Pet, and Adoption data)
* **Caching:** Redis (for caching frequently accessed data)
* **Message Queue:** NATS (for asynchronous event publishing and consumption)
* **Containerization:** Docker & Docker Compose
* **Email Sending:** SMTP (via Go's `net/smtp` package)
* **Testing:** Go's built-in `testing` package (for unit tests)

## 3. Project Architecture

The project follows a microservices architecture consisting of the following services:

* **API Gateway (`api-gateway`):** Exposes RESTful APIs to external clients and routes requests to the appropriate backend gRPC services. Handles initial request validation and potentially JWT authentication.
* **User Service (`user-service`):** Manages user registration, login (issuing JWTs), profile retrieval, and updates. Uses MongoDB for storage and Redis for caching.
* **Pet Service (`pet-service`):** Manages pet listings, including details, stock (implicitly via adoption status), and status updates. Uses MongoDB for storage and Redis for caching.
* **Adoption Service (`adoption-service`):** Handles the creation and management of adoption applications. Publishes events to NATS upon application creation and status changes. Uses MongoDB for storage and Redis for caching.
* **Notification Service (`notification-service`):** Subscribes to NATS events from the `adoption-service` and sends email notifications to users (e.g., application received, application approved/rejected). It uses gRPC clients to fetch user and pet details for email content.

**(Optional: You can add a simple architecture diagram here if you create one, e.g., using draw.io or a text-based tool like Mermaid via a Markdown extension if your Git platform supports it).**

## 4. How to Run Locally (with Docker)

This project is designed to be run using Docker and Docker Compose for a consistent development environment.

**Prerequisites:**
* Docker installed and running.
* Docker Compose installed (usually comes with Docker Desktop).
* Git (to clone the repository).
* A configured `.env` file at the project root (see `.env.example`).

**Steps:**

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/zhandarbeks/petstore-final-project.git](https://github.com/zhandarbeks/petstore-final-project.git)
    cd petstore-final-project
    ```

2.  **Create your environment configuration file:**
    Copy the example environment file and customize it with your settings (especially for SMTP if you want to test email sending, and JWT secrets).
    ```bash
    cp .env.example .env
    ```
    Open `.env` in a text editor and update the placeholder values, particularly:
    * `JWT_SECRET_KEY` (make this a strong, unique random string, ensure it's the same for `user-service` and `api-gateway` if the gateway validates tokens).
    * `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SENDER_EMAIL` (for the `notification-service` to send emails). For Gmail, use an "App Password".

3.  **Build and run all services using Docker Compose:**
    From the project root directory (`petstore-final-project`), run:
    ```bash
    docker-compose up --build
    ```
    * The `--build` flag ensures images are rebuilt if there are changes to Dockerfiles or source code.
    * To run in detached mode (in the background), add the `-d` flag: `docker-compose up --build -d`.

4.  **Services will be available at:**
    * **API Gateway (REST):** `http://localhost:8080` (or as configured by `API_GATEWAY_HOST_PORT` in `.env`)
    * **User Service (gRPC):** `localhost:50051` (or as configured by `USER_SERVICE_HOST_PORT`)
    * **Pet Service (gRPC):** `localhost:50052` (or as configured by `PET_SERVICE_HOST_PORT`)
    * **Adoption Service (gRPC):** `localhost:50053` (or as configured by `ADOPTION_SERVICE_HOST_PORT`)
    * **NATS Monitoring:** `http://localhost:8222`
    * **MongoDB:** Accessible on `localhost:27017` from your host machine (e.g., via MongoDB Compass).
    * **Redis:** Accessible on `localhost:6379` from your host machine (e.g., via `redis-cli`).

5.  **To stop all services:**
    ```bash
    docker-compose down
    ```
    To stop and remove volumes (deletes data):
    ```bash
    docker-compose down -v
    ```

## 5. How to Run Tests

Unit tests are provided for the usecase layers of the services.

1.  **Navigate to the specific service directory:**
    For example, to test the `user-service`:
    ```bash
    cd user-service
    ```
    Or for the `pet-service`:
    ```bash
    cd pet-service
    ```
    (And similarly for `adoption-service` and `notification-service`).

2.  **Run the tests for that service:**
    ```bash
    go test -v ./...
    ```
    The `-v` flag enables verbose output. The `./...` pattern runs tests in the current directory and all its subdirectories.

## 6. Description of gRPC Endpoints

The core backend services expose the following gRPC endpoints. These are typically consumed by the API Gateway or by other services internally.

*(You can use `grpcurl -plaintext <service_host>:<service_port> describe <package.ServiceName>` to get detailed descriptions if server reflection is enabled).*

### 6.1. User Service (`user.UserService` on port 50051)

* **`RegisterUser(RegisterUserRequest) returns (UserResponse)`**
    * Registers a new user.
    * Request: `username`, `email`, `password`, `full_name`.
    * Response: Created `User` object.
* **`LoginUser(LoginUserRequest) returns (LoginUserResponse)`**
    * Authenticates an existing user.
    * Request: `email`, `password`.
    * Response: `User` object and an `access_token` (JWT).
* **`GetUser(GetUserRequest) returns (UserResponse)`**
    * Retrieves a user's profile by their ID.
    * Request: `user_id`.
    * Response: `User` object.
* **`UpdateUserProfile(UpdateUserProfileRequest) returns (UserResponse)`**
    * Updates a user's profile (username, full_name).
    * Request: `user_id`, optional `username`, optional `full_name`.
    * Response: Updated `User` object.
* **`DeleteUser(DeleteUserRequest) returns (EmptyResponse)`**
    * Deletes a user account by ID.
    * Request: `user_id`.
    * Response: Empty.

### 6.2. Pet Service (`pet.PetService` on port 50052)

* **`CreatePet(CreatePetRequest) returns (PetResponse)`**
    * Adds a new pet listing.
    * Request: `name`, `species`, `breed`, `age`, `description`, `listed_by_user_id`, `image_urls`.
    * Response: Created `Pet` object.
* **`GetPet(GetPetRequest) returns (PetResponse)`**
    * Retrieves a pet's details by its ID.
    * Request: `pet_id`.
    * Response: `Pet` object.
* **`UpdatePet(UpdatePetRequest) returns (PetResponse)`**
    * Updates an existing pet's details.
    * Request: `pet_id`, optional `name`, `species`, `breed`, `age`, `description`, `image_urls`.
    * Response: Updated `Pet` object.
* **`DeletePet(DeletePetRequest) returns (EmptyResponse)`**
    * Deletes a pet listing by ID.
    * Request: `pet_id`.
    * Response: Empty.
* **`ListPets(ListPetsRequest) returns (ListPetsResponse)`**
    * Lists pets with pagination and optional filters.
    * Request: optional `page`, `limit`, `species_filter`, `status_filter`.
    * Response: List of `Pet` objects, `total_count`, `page`, `limit`.
* **`UpdatePetAdoptionStatus(UpdatePetAdoptionStatusRequest) returns (PetResponse)`**
    * Updates a pet's adoption status and adopter ID.
    * Request: `pet_id`, `new_status`, `adopter_user_id`.
    * Response: Updated `Pet` object.

### 6.3. Adoption Service (`adoption.AdoptionService` on port 50053)

* **`CreateAdoptionApplication(CreateAdoptionApplicationRequest) returns (AdoptionApplicationResponse)`**
    * Creates a new adoption application for a pet by a user.
    * Request: `user_id`, `pet_id`, `application_notes`.
    * Response: Created `AdoptionApplication` object.
* **`GetAdoptionApplication(GetAdoptionApplicationRequest) returns (AdoptionApplicationResponse)`**
    * Retrieves an adoption application by its ID.
    * Request: `application_id`.
    * Response: `AdoptionApplication` object.
* **`UpdateAdoptionApplicationStatus(UpdateAdoptionApplicationStatusRequest) returns (AdoptionApplicationResponse)`**
    * Updates the status of an adoption application (e.g., to APPROVED, REJECTED).
    * Request: `application_id`, `new_status`, `review_notes`.
    * Response: Updated `AdoptionApplication` object.
* **`ListUserAdoptionApplications(ListUserAdoptionApplicationsRequest) returns (ListAdoptionApplicationsResponse)`**
    * Lists all adoption applications for a specific user, with pagination and optional status filter.
    * Request: `user_id`, optional `page`, `limit`, `status_filter`.
    * Response: List of `AdoptionApplication` objects, `total_count`, `page`, `limit`.

## 7. List of Implemented Features (Meeting Project Requirements)

* **Clean Architecture:** Applied across all backend microservices (`user-service`, `pet-service`, `adoption-service`, `notification-service`).
* **gRPC Endpoints:** A total of 15 gRPC endpoints implemented across the services, exceeding the minimum requirement of 12.
* **Message Queue (NATS):**
    * `adoption-service` publishes events (`adoption.application.created`, `adoption.application.status.updated`) to NATS.
    * `notification-service` consumes these events from NATS.
* **Databases and Caches:**
    * **MongoDB:** Used as the primary persistent database for `user-service`, `pet-service`, and `adoption-service`.
    * **Redis:** Implemented as a caching layer for read-heavy operations in `user-service`, `pet-service`, and `adoption-service`.
    * **Migrations/Schema Setup:** Index creation is handled during repository initialization for MongoDB collections.
    * **Transactions:** *(Specify where and how you implemented the required database transaction, e.g., "Implemented a MongoDB transaction in the adoption-service when an application status is updated to also write to an audit log atomically.")*
* **Sending Emails:**
    * `notification-service` is responsible for sending emails (e.g., adoption application received, status updates) using SMTP.
* **Testing:**
    * Unit tests (using Go's `testing` package) are implemented for the usecase layers of `user-service`, `pet-service`, `adoption-service`, and `notification-service`, utilizing mocks for dependencies.
* **Containerization:** All services are containerized using Docker and orchestrated with Docker Compose for local development and deployment.
* **API Gateway (`api-gateway`):** Provides a RESTful interface using Gin, translating HTTP requests to gRPC calls to backend services.
* **Authentication:** JWT-based authentication implemented in `user-service` (token generation) and the API Gateway can be configured to validate these tokens for protected routes.

*(Remember to update the "Transactions" part once you've implemented it!)*
