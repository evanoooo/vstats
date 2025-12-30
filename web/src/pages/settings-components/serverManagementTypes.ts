/**
 * Server Management Section Type Definitions
 */

import type { ServerLabel } from './types';

// Edit form state for server editing
export interface EditForm {
  name: string;
  location: string;
  provider: string;
  tag: string;
  price_amount: string;
  price_period: 'month' | 'quarter' | 'year';
  price_currency: string;
  purchase_date: string;
  auto_renew: boolean;
  tip_badge: string;
  notes: string;
  group_values: Record<string, string>;
  labels: ServerLabel[];
  sale_status: '' | 'rent' | 'sell';
  sale_contact_url: string;
  // Traffic settings
  traffic_limit_gb: string;
  traffic_threshold_type: 'sum' | 'max' | 'up' | 'down';
  traffic_reset_day: number;
}

// Default edit form values
export const DEFAULT_EDIT_FORM: EditForm = {
  name: '',
  location: '',
  provider: '',
  tag: '',
  price_amount: '',
  price_period: 'month',
  price_currency: 'USD',
  purchase_date: '',
  auto_renew: false,
  tip_badge: '',
  notes: '',
  group_values: {},
  labels: [],
  sale_status: '',
  sale_contact_url: '',
  traffic_limit_gb: '',
  traffic_threshold_type: 'sum',
  traffic_reset_day: 1,
};

// Server Management Section props
export interface ServerManagementSectionProps {
  token: string | null;
  isZh: boolean;
}

