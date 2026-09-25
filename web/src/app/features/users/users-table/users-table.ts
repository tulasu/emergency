import { Component, input, output } from '@angular/core';
import { FormArray, ReactiveFormsModule } from '@angular/forms';
import { TbIcon } from '../../../shared/ui/icon/icon';
import { TbInput } from '../../../shared/ui/input/input';
import { UserRowGroup } from '../users-form';

@Component({
  selector: 'tb-users-table',
  imports: [ReactiveFormsModule, TbIcon, TbInput],
  templateUrl: './users-table.html',
  styleUrl: './users-table.css',
})
export class UsersTable {
  readonly users = input.required<FormArray<UserRowGroup>>();
  readonly add = output();
  readonly remove = output<number>();

  rowGroup(index: number): UserRowGroup {
    return this.users().at(index);
  }
}
