import { Component, HostListener, inject, signal } from '@angular/core';
import { Router, RouterLink, RouterLinkActive } from '@angular/router';
import { AuthStore } from '../../auth/auth.store';
import { NavHistory } from '../../nav/nav-history';
import { TbIcon } from '../../../shared/ui/icon/icon';

@Component({
  selector: 'tb-sidebar',
  imports: [RouterLink, RouterLinkActive, TbIcon],
  templateUrl: './sidebar.html',
  styleUrl: './sidebar.css',
})
export class Sidebar {
  private readonly store = inject(AuthStore);
  private readonly router = inject(Router);
  readonly nav = inject(NavHistory);

  readonly profileOpen = signal(false);

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
