/** Color utility functions for entity detail views */

export function getStepStatusColor(status: string): string {
	switch (status) {
		case 'completed':
			return '#10b981';
		case 'in_progress':
			return '#f59e0b';
		default:
			return '#d1d5db';
	}
}

export function getMilestoneColor(milestone?: string): string {
	if (!milestone) return '#6b7280';
	const colors: Record<string, string> = {
		M1: '#10b981', M2: '#3b82f6', M3: '#8b5cf6',
		M4: '#f59e0b', M5: '#ef4444', M6: '#ec4899', M7: '#06b6d4'
	};
	return colors[milestone] || '#6b7280';
}

export function getCategoryColor(category: string): string {
	switch (category) {
		case 'object': return '#3b82f6';
		case 'dictionary': return '#8b5cf6';
		case 'value-object': return '#f59e0b';
		default: return '#6b7280';
	}
}

export function getCriterionStatusIcon(status: string): string {
	switch (status) {
		case 'pass': return '\u2713';
		case 'fail': return '\u2717';
		default: return '?';
	}
}

export function getCriterionStatusColor(status: string): string {
	switch (status) {
		case 'pass': return '#10b981';
		case 'fail': return '#ef4444';
		default: return '#6b7280';
	}
}
