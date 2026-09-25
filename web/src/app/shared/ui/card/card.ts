import { Component } from '@angular/core';

@Component({
  selector: '[tbCard]',
  template: `<ng-content />`,
  styleUrl: './card.css',
  host: {
    class: 'tb-card',
  },
})
export class TbCard {}
