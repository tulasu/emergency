import { Component, effect, input, signal } from '@angular/core';

@Component({
  selector: 'button[tbButton]',
  template: `
    @if (showSpin()) {
      <span class="tb-btn__spin" aria-hidden="true"></span>
    }
    <span class="tb-btn__label" [class.tb-btn__label--hidden]="showSpin()">
      <ng-content />
    </span>
  `,
  styleUrl: './button.css',
  host: {
    '[class]': '"tb-btn tb-btn--" + variant() + " tb-btn--" + width()',
    '[class.tb-btn--busy]': 'busy()',
    '[disabled]': 'disabled() || busy()',
    '[attr.aria-busy]': 'busy() ? true : null',
  },
})
export class TbButton {
  readonly variant = input<'primary' | 'secondary' | 'ghost' | 'danger'>('primary');
  readonly width = input<'auto' | 'block'>('auto');
  readonly disabled = input(false);
  readonly busy = input(false);
  readonly showSpin = signal(false);

  constructor() {
    effect((onCleanup) => {
      if (!this.busy()) {
        this.showSpin.set(false);
        return;
      }
      const id = setTimeout(() => this.showSpin.set(true), 200);
      onCleanup(() => clearTimeout(id));
    });
  }
}
