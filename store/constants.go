// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

const (
	ChannelExistsError = "store.sql_channel.save_channel.exists.app_error"

	UserSearchOptionNamesOnly           = "names_only"
	MISSING_LTI_ACCOUNT_ERROR  = "store.sql_user.get_by_lti.missing_account.app_error"
	UserSearchOptionNamesOnlyNoFullName = "names_only_no_full_name"
	UserSearchOptionAllNoFullName       = "all_no_full_name"
	UserSearchOptionAllowInactive       = "allow_inactive"

	FeatureTogglePrefix = "feature_enabled_"
)
