import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { Sidebar } from '../sidebar/sidebar';

@Component({
  selector: 'tb-app-shell',
  imports: [RouterOutlet, Sidebar],
  template: `
    <div class="shell">
      <tb-sidebar />
      <main class="shell__main">
        <router-outlet />
      </main>
    </div>
  `,
  styles: `
    .shell {
      min-height: 100vh;
      display: flex;
      gap: var(--space-lg);
      padding: 24px;
      background: var(--color-bg);
    }
    .shell__main {
      flex: 1;
      min-width: 0;
    }
  `,
})
export class AppShell {}
