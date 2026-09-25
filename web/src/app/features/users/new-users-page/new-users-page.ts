import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { TbCard } from '../../../shared/ui/card/card';
import { TbIcon } from '../../../shared/ui/icon/icon';

@Component({
  selector: 'tb-new-users-page',
  imports: [RouterLink, TbCard, TbIcon],
  templateUrl: './new-users-page.html',
  styleUrl: './new-users-page.css',
})
export class NewUsersPage {}
