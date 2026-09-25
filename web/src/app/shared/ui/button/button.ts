import { Component, input } from '@angular/core';

@Component({
  selector: 'button[tbButton]',
  template: `<ng-content />`,
  styleUrl: './button.css',
  host: {
    '[class]': '"tb-btn tb-btn--" + variant() + " tb-btn--" + width()',
    '[disabled]': 'disabled()',
  },
})
export class TbButton {
  readonly variant = input<'primary' | 'secondary' | 'ghost' | 'danger'>('primary');
  readonly width = input<'auto' | 'block'>('auto');
  readonly disabled = input(false);
}
