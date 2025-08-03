# ADR-008: Examples & Learning Journey Strategy

**Accepted** - Implemented ✅
**Implementation Date**: August 2025
**Authors**: Community Product Manager, Go OSS Reviewer  
**Related**: [ADR-007: Boilerplate Reduction & Router Composition](./ADR-007-boilerplate-reduction-router-composition.md)

## Summary

This ADR defines a comprehensive examples and documentation strategy to help developers go from 0 to 1 with TypedHTTP, flattening the learning curve and maximizing community adoption.

## Context

### Current State Analysis

**Strengths**:
- Excellent technical foundation with 52% boilerplate reduction
- Comprehensive architecture examples showing real-world patterns
- Strong OpenAPI integration with automatic generation
- Type-safe approach aligns with Go community values

**Adoption Barriers Identified**:
- High barrier to entry (current "simple" example is 107 lines)
- Missing progressive learning path
- No migration guides from popular frameworks (Gin, Echo, Chi)
- Lack of performance benchmarks for trust building
- Complex examples before basic patterns

### Strategic Insights

**Community Product Manager Analysis**:
- Target 50+ engineer organizations require both simple onboarding AND enterprise patterns
- Time-to-first-success must be < 5 minutes for evaluation phase
- Migration anxiety is a major adoption blocker
- Performance transparency builds community trust

**Go OSS Reviewer Analysis**:
- Go community values progressive complexity and clear idioms
- Copy-paste effectiveness is critical for adoption
- Benchmarks vs existing frameworks establish credibility
- Testing patterns in examples teach best practices

## Decision

We will implement a **Progressive Learning Journey** that restructures examples into a clear 0-to-1 path while maintaining advanced enterprise patterns.

### New Examples Structure

```
examples/
├── 01-quickstart/              # <5min: Single file, instant success
├── 02-fundamentals/            # 5-15min: Real CRUD with testing
├── 03-intermediate/            # Enhanced current simple/
├── 04-production/              # 30min-2hr: Enterprise deployment
├── 05-advanced/                # Renamed comprehensive-architecture/
├── migration/                  # Framework migration guides
│   ├── from-gin/
│   ├── from-echo/
│   └── from-chi/
├── recipes/                    # Specific use case patterns
│   ├── auth-patterns/
│   ├── file-uploads/
│   ├── testing-strategies/
│   └── performance/
├── benchmarks/                 # Performance vs other frameworks
│   ├── vs-gin/
│   ├── vs-echo/
│   └── vs-chi/
└── integrations/               # Ecosystem confidence
    ├── gorm-database/
    ├── redis-caching/
    ├── temporal-workflows/
    └── prometheus-metrics/
```

### Success Criteria

**Developer Velocity Metrics**:
- Time-to-first-success: < 5 minutes from discovery to running API
- Copy-paste effectiveness: 90% of developers can adapt examples without modification
- Learning progression completion: Track through analytics

**Community Trust Indicators**:
- Performance transparency: Concrete benchmarks vs Gin/Echo/Chi
- Migration confidence: Clear, tested migration paths
- Production readiness: Docker, K8s, CI/CD examples

**Ecosystem Integration**:
- Framework compatibility with popular Go libraries
- Clear integration patterns and examples
- Tooling support demonstrations

## Implementation Plan

### Phase 1: Foundation (Week 1-2)

**Critical Path Examples**:

1. **01-quickstart/** - Instant Gratification
   ```go
   // Target: 15 lines, single file
   package main
   import (
       "context"
       "github.com/pavelpascari/typedhttp/pkg/typedhttp"
   )
   type User struct { Name string `json:"name"` }
   func GetUser(ctx context.Context, req struct{ID string `path:"id"`}) (User, error) {
       return User{Name: "Hello " + req.ID}, nil
   }
   func main() {
       router := typedhttp.NewRouter()
       typedhttp.GET(router, "/users/{id}", GetUser)
       http.ListenAndServe(":8080", router)
   }
   ```

2. **migration/from-gin/** - Adoption Catalyst
   - Side-by-side feature comparison
   - Step-by-step migration guide
   - Before/after code samples
   - Benefits quantification

3. **benchmarks/** - Trust Building
   - Latency, throughput, memory usage vs competitors
   - Reproducible benchmark scripts
   - Performance optimization guides

### Phase 2: Production Readiness (Week 3-4)

1. **02-fundamentals/** - Real CRUD (~50 lines)
   - In-memory storage for simplicity
   - All CRUD operations with validation
   - Comprehensive testing patterns
   - Docker deployment ready

2. **04-production/** - Enterprise Ready
   - Database integration (GORM)
   - Authentication and authorization
   - Logging and monitoring
   - Kubernetes manifests

3. **recipes/** - Common Patterns
   - JWT authentication
   - File upload handling
   - Testing strategies
   - Performance optimization

### Phase 3: Community Building (Ongoing)

1. **Enhanced Documentation**
   - Progressive disclosure in README.md
   - Video walkthroughs for visual learners
   - Clear value propositions
   - Feature comparison matrices

2. **Community Outreach**
   - HackerNews/Reddit launch strategy
   - Conference talk submissions
   - Integration with Awesome Go
   - Corporate case studies

## Expected Outcomes

### Immediate Benefits (Month 1)
- Reduced time-to-first-success from ~30 minutes to <5 minutes
- Clear migration paths reduce framework switching anxiety
- Performance benchmarks establish technical credibility
- Progressive complexity enables both rapid evaluation and deep adoption

### Long-term Impact (Months 2-6)
- Increased GitHub stars and community engagement
- Production adoption by 50+ engineer organizations
- Ecosystem integrations with popular Go libraries
- Thought leadership in Go HTTP framework space

### Metrics to Track
- GitHub stars growth (target: 500+ in Q1)
- Example completion rates via analytics
- Community questions shifting from "how to start" to "how to optimize"
- Production deployment stories and case studies

## Technical Considerations

### Code Quality Standards
- All examples follow Go idioms and best practices
- Comprehensive test coverage demonstrates testing patterns
- Error handling showcases TypedHTTP error types
- Documentation comments explain architectural decisions

### Performance Characteristics
- Benchmark all examples against memory and CPU usage
- Demonstrate TypedHTTP's competitive performance
- Show optimization techniques and patterns
- Provide performance tuning guides

### Maintenance Strategy
- Automated testing of all examples in CI/CD
- Version compatibility testing across Go versions
- Regular updates with new TypedHTTP features
- Community contribution guidelines for new examples

## Alternative Approaches Considered

### Single Complex Example
**Rejected**: High cognitive load prevents evaluation and adoption

### Framework-Specific Migration Only
**Rejected**: Misses opportunity to showcase unique TypedHTTP advantages

### Documentation-Heavy Approach
**Rejected**: Go community prefers working code over extensive prose

## Implementation Notes

### Phase 1 Priority Order
1. ADR-008 documentation (this document)
2. 01-quickstart/ for immediate impact
3. migration/from-gin/ for adoption acceleration
4. Enhanced README.md with progressive disclosure

### Success Measurement
- Analytics on example repository clones/views
- GitHub issues shifting from setup to optimization questions
- Community adoption stories and case studies
- Conference talk acceptance and feedback

## Conclusion

This comprehensive examples strategy transforms TypedHTTP from "technically impressive but intimidating" to "immediately adoptable and scalable." By providing a clear learning progression from 5-minute evaluation to enterprise deployment, we enable both rapid adoption and deep integration.

The strategy addresses the core tension between simplicity and sophistication by using progressive disclosure - developers can start simple and grow into advanced patterns as needed. This approach maximizes both community growth and enterprise adoption potential.

---

**Next Steps**: Begin implementation with Phase 1 examples, starting with 01-quickstart/ and migration/from-gin/ for maximum adoption impact.