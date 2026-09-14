# Organization & Party Core

This guide details the design and usage of the **Organization Aggregate**, **Party Aggregate**, and **Address & Contact Primitives** in Mergiate Core.

---

## 1. Organization Aggregate (Hierarchical Tree)

In Mergiate, a **Tenant** can contain multiple legal entities, branches, and departments. An `Organization` represents an internal operational unit.

### Hierarchy & Anti-Cycle Protection
Organizations form a tree structure using `parent_id`:
```
Holding Company (company)
  ├── Surabaya Branch (branch)
  │     ├── Finance Dept (department)
  │     └── Warehouse A (location)
  └── Jakarta Branch (branch)
```

- **Invariant**: Cyclic hierarchies (e.g. setting an organization's parent to itself or one of its descendants) are rejected by `CheckCircularReference`.
- **JSON Tree**: The `GET /api/v1/organizations/tree` endpoint recursively constructs the nested tree inside the `children` property.

---

## 2. Party Aggregate (Unified Business Actors)

Mergiate eliminates the common anti-pattern of separate, disjointed tables for `customers`, `suppliers`, and `partners`.

### Why "Party"?
A real-world commercial partner (e.g., `PT Sumber Makmur`) is often **both** a customer (they buy your products) and a supplier (they supply raw materials to you).

Instead of duplicating master records:
- **`Party`**: Represents the actor (`type: "organization"` or `"person"`), holding identity, legal name, and tax ID (`NPWP`).
- **`PartyRole`**: Represents active commercial roles (`customer`, `supplier`, `partner`, `employee`, `agent`).

```
Party: PT Sumber Makmur (type: organization)
  ├── Role: Customer
  ├── Role: Supplier
  ├── Address: Surabaya Operational Office
  └── Contact: Finance Email (accounting@sumbermakmur.com)
```

### Querying Parties by Role
You can filter parties by role via query parameters:
```bash
GET /api/v1/parties?role=supplier
GET /api/v1/parties?role=customer
```

---

## 3. Reusable Address & Contact Primitives

Addresses and contacts are universal primitives linked to parties or organizations:

- **Address**: Stores structured location data (`line1`, `city`, `state`, `postal_code`, `country_code`, `latitude`, `longitude`) and is linked with an `address_type` (`office`, `billing`, `shipping`, `warehouse`).
- **Contact**: Stores communication endpoints (`email`, `phone`, `mobile`, `website`) with flags like `is_primary` and `verified_at`.

---

## 4. API Endpoints Reference

### Organizations
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/organizations` | Create an organization |
| `GET` | `/api/v1/organizations` | List organizations flat |
| `GET` | `/api/v1/organizations/tree` | Get nested hierarchy tree |
| `GET` | `/api/v1/organizations/{id}` | Get organization by ID |
| `PUT` | `/api/v1/organizations/{id}` | Update organization & parent |

### Parties
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/parties` | Create party with optional initial roles |
| `GET` | `/api/v1/parties` | List parties with pagination & filters (`?role=...&type=...`) |
| `GET` | `/api/v1/parties/{id}` | Get party details (with roles, addresses, contacts) |
| `PUT` | `/api/v1/parties/{id}` | Update party details |
| `POST` | `/api/v1/parties/{id}/roles` | Assign a new role |
| `DELETE` | `/api/v1/parties/{id}/roles/{roleId}` | Remove a role |
| `POST` | `/api/v1/parties/{id}/addresses` | Add and link an address |
| `POST` | `/api/v1/parties/{id}/contacts` | Add a contact |
