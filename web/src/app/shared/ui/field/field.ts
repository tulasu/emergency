import { Component, input } from '@angular/core';

@Component({
  selector: 'tb-field',
  template: `
    <span class="tb-field__label">{{ label() }}</span>
    <ng-content />
  `,
  styleUrl: './field.css',
})
export class TbField {
  readonly label = input.required<string>();
}
