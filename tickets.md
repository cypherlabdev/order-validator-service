# order-validator-service - Ticket Tracking

## Service Overview
**Repository**: github.com/cypherlabdev/order-validator-service
**Purpose**: Validate orders and orchestrate sagas with Temporal
**Implementation Status**: 60% complete (validation logic exists, gRPC server missing)
**Language**: Go 1.21+

## Existing Asana Tickets
### 1. [1211394356065986] ENG-78: Order Validator
**Task ID**: 1211394356065986 | **ENG Field**: ENG-78
**URL**: https://app.asana.com/0/1211254851871080/1211394356065986
**Assignee**: sj@cypherlab.tech
**Dependencies**: ⬆️ ENG-82 (order-book), ⬇️ None (saga coordinator)

## Tickets to Create
1. **[NEW] Implement gRPC Server (P0)** - Server setup missing
2. **[NEW] Implement Saga Workflows with Temporal (P0)** - 2PC coordination
3. **[NEW] Add Validation Rules Engine (P1)** - Configurable rules
