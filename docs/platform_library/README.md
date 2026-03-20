# Platform Library

The platform library should be one of the core moats of AresSim.

The goal is not just breadth. The goal is a library that is:
- globally useful
- operationally credible
- reviewable
- maintainable
- explicit about uncertainty

## Standard

Every platform entry should aim to answer four questions well:
- What is it?
- Who actually operates it?
- What can it realistically do in combat?
- What does it cost to lose?

That means the library should optimize for:
- correct operator mapping by country and service
- credible range, speed, endurance, survivability, and sensing
- believable loadouts and mission fits
- real-world replacement / strategic / economic / human-loss value inputs

## Classification Model

The current hierarchy is:
- `domain`
- `form`
- `general_type`
- `specific_type`

This is intentionally compact. We should resist adding dozens of one-off fields when a stronger controlled vocabulary is enough.

The recent expansion of `general_type` is meant to cover real global force-structure gaps:
- electronic-attack aircraft
- ASW / heavy-lift / CSAR helicopters
- strike / EW / attritable unmanned aircraft
- missile boats and offshore patrol vessels
- coastal and special-mission submarines
- ballistic-missile launchers, coastal-defense missile units, radar / early-warning units, and C-RAM
- signals-intelligence and airbase-support units

These classes are important because modern militaries are not made up only of fighters, tanks, destroyers, and SAM batteries.

## Required Data Quality

For a platform to be considered `credible`, it should eventually carry:
- canonical platform name and variant
- operator countries
- service entry year
- mission class / platform class
- movement performance:
  - max speed
  - cruise speed
  - practical combat radius / range proxy
- sensor quality:
  - detection range proxy
  - radar cross section proxy if relevant
- survivability proxy
- fuel / endurance proxy
- authorized personnel
- default and alternative weapon configurations
- replacement cost
- strategic value
- economic value if it is dual-use or infrastructure

## Source Hierarchy

We should prefer sources in this order:
1. official manufacturer data
2. official government / military fact sheets and budget documents
3. major defense reference publications
4. reputable open-source analysis
5. informed estimates, explicitly marked as such

We should avoid basing entries on:
- random blogs
- image-board style aggregations
- unsourced wiki values copied blindly

## Confidence Model

We should add a confidence workflow later, even if we do not expose it in the UI yet.

The model now carries first-pass provenance fields:
- `data_confidence`
- `source_basis`
- `source_notes`
- `source_links`

Current normalized values:
- `data_confidence`: `high`, `medium`, `low`, `heuristic`
- `source_basis`: `manufacturer`, `government_or_official`, `reputable_analysis`, `estimated`, `heuristic`

The moat is not just data volume. It is trustworthy data with visible uncertainty.

## Authoring Workflow

Recommended process for new libraries:
1. choose a country and service slice
2. enumerate the actual order-of-battle relevant platforms
3. map each platform to one of the controlled platform classes
4. fill operator and variant data
5. author default combat-performance proxies
6. author realistic mission loadouts
7. set cost / strategic / economic values
8. review for consistency against peers in the same class

## Priority Backlog

Near-term high-value areas:
- Israel, Iran, and U.S. platform depth
- Gulf-state air-defense, fighter, MPA, and naval forces
- ballistic-missile and drone force libraries
- enablers:
  - tankers
  - AEW
  - SIGINT
  - EW
  - airbase support
- naval posture assets:
  - missile boats
  - OPVs
  - coastal submarines
  - mine warfare

## Non-Goals

The library should not pretend to model:
- exact classified radar performance
- perfect missile PK tables
- hidden national modifications when they are not supportable from sources

When uncertain, prefer a bounded, reviewable proxy over false precision.
