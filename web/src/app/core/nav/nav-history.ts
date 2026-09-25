import { Location } from '@angular/common';
import { Injectable, inject, signal } from '@angular/core';
import { ActivatedRouteSnapshot, NavigationEnd, NavigationStart, Router } from '@angular/router';

export const skipHistory = { skipHistory: true } as const;

@Injectable({ providedIn: 'root' })
export class NavHistory {
  private readonly router = inject(Router);
  private readonly location = inject(Location);

  private stack: string[] = [];
  private trigger: NavigationStart['navigationTrigger'] = 'imperative';
  private replaceUrl = false;

  readonly canGoBack = signal(false);

  constructor() {
    this.router.events.subscribe((event) => {
      if (event instanceof NavigationStart) {
        this.trigger = event.navigationTrigger;
        this.replaceUrl = !!this.router.getCurrentNavigation()?.extras.replaceUrl;
        return;
      }
      if (event instanceof NavigationEnd) {
        this.onEnd(event.urlAfterRedirects);
      }
    });
  }

  back(): void {
    if (!this.canGoBack()) {
      return;
    }
    this.location.back();
  }

  private onEnd(url: string): void {
    if (hasSkipHistory(this.router.routerState.snapshot.root)) {
      this.stack = [];
      this.canGoBack.set(false);
      return;
    }

    const path = normalize(url);
    if (this.trigger === 'popstate') {
      const idx = this.stack.lastIndexOf(path);
      if (idx >= 0) {
        this.stack = this.stack.slice(0, idx + 1);
      } else {
        this.stack.push(path);
      }
    } else if (this.replaceUrl && this.stack.length > 0) {
      this.stack[this.stack.length - 1] = path;
    } else if (this.stack[this.stack.length - 1] !== path) {
      this.stack.push(path);
    }

    this.canGoBack.set(this.stack.length > 1);
  }
}

function normalize(url: string): string {
  const path = url.split(/[?#]/)[0];
  return path === '' ? '/' : path;
}

function hasSkipHistory(route: ActivatedRouteSnapshot): boolean {
  return route.data['skipHistory'] === true || route.children.some(hasSkipHistory);
}
