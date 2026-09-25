import { Component, inject } from '@angular/core';
import { AuthStore } from '../../core/auth/auth.store';

@Component({
  selector: 'tb-home-page',
  template: `
    <section class="home">
      <h1 class="page-title">Токенoeжки</h1>
      <p class="page-sub">Вы вошли как {{ store.user()?.login }}</p>
    </section>
  `,
  styles: `
    .home { padding: 24px 8px; display: flex; flex-direction: column; gap: 8px; }
  `,
})
export class HomePage {
  readonly store = inject(AuthStore);
}
