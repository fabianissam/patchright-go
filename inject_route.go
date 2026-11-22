package playwright

import "strings"

// patchrightInjectRouteHandler marks document navigations so the patched
// driver can inject init scripts without Runtime.enable. Matching
// patchright-python's install_inject_route.
func patchrightInjectRouteHandler(route Route) {
	req := route.Request()
	if req != nil && req.ResourceType() == "document" && strings.HasPrefix(req.URL(), "http") {
		if err := route.Fallback(RouteFallbackOptions{PatchrightInitScript: Bool(true)}); err == nil {
			return
		}
	}
	_ = route.Fallback()
}

func (b *browserContextImpl) installInjectRoute() error {
	if b.routeInjecting {
		return nil
	}
	if err := b.Route("**/*", patchrightInjectRouteHandler); err != nil {
		return err
	}
	b.routeInjecting = true
	return nil
}

func (p *pageImpl) installInjectRoute() error {
	if p.routeInjecting {
		return nil
	}
	if p.browserContext != nil && p.browserContext.routeInjecting {
		return nil
	}
	if err := p.Route("**/*", patchrightInjectRouteHandler); err != nil {
		return err
	}
	p.routeInjecting = true
	return nil
}
