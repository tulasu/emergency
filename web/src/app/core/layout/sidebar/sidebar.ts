import { Component, HostListener, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink } from '@angular/router';
import { filter, map, startWith } from 'rxjs';
import { AuthStore } from '../../auth/auth.store';
import { NavHistory } from '../../nav/nav-history';
import { TbIcon } from '../../../shared/ui/icon/icon';

@Component({
  selector: 'tb-sidebar',
  imports: [RouterLink, TbIcon],
  templateUrl: './sidebar.html',
  styleUrl: './sidebar.css',
})
export class Sidebar {
  private readonly store = inject(AuthStore);
  private readonly router = inject(Router);
  readonly nav = inject(NavHistory);

  readonly profileOpen = signal(false);
  readonly usersActive = toSignal(
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
      map((event) => event.urlAfterRedirects.startsWith('/users')),
      startWith(this.router.url.startsWith('/users')),
    ),
    { initialValue: this.router.url.startsWith('/users') },
  );

  toggleProfile(event: Event): void {
    event.stopPropagation();
    this.profileOpen.update((open) => !open);
  }

  @HostListener('document:click')
  closeProfile(): void {
    this.profileOpen.set(false);
  }

  @HostListener('document:keydown.escape')
  closeOnEsc(): void {
    this.profileOpen.set(false);
  }

  async logout(): Promise<void> {
    this.profileOpen.set(false);
    await this.store.logout();
    await this.router.navigateByUrl('/login');
  }
}
