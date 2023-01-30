#! /usr/bin/env perl

use strict;
use File::Basename;

# sudo dnf install -y perl-Text-ASCIITable.noarch
use Text::ASCIITable;

# Usage:
# ./result-summary.pl ./RESULTS/20230130-103747 ./RESULTS/20230130-113758

my $debug = 0;

sub average {
    return 0 unless @_;
    return int(sum(@_) / @_);
}

sub sum {
    return 0 unless @_;
    my $sum;
    $sum += $_ for @_;
    return $sum;
}

sub parse_result_dir {
    my ($dir) = @_;

    my $result = {
	traffic_type => basename($dir),
	hits => [],
	rps => [],
	connection_errors => [],
	status_errors => [],
	parser_errors => [],
    };

    for my $rfile (glob("$dir/*.stdout")) {
	open(FH, '<', $rfile);
	while (<FH>) {
	    if (/Hits: (\d+), (\d+)/) {
		print STDERR "hits: $1 rps: $2 " if $debug;
		push(@{$result->{hits}}, "$1");
		push(@{$result->{rps}}, "$2");
	    } elsif (/Errors connection: (\d+), status: (\d+), parser: (\d+)/) {
		print STDERR "connection_errors: $1 status_errors: $2 parser_errors: $3\n" if $debug;
		push(@{$result->{connection_errors}}, "$1");
		push(@{$result->{status_errors}}, "$2");
		push(@{$result->{parser_errors}}, "$3");
	    } elsif (/Time/) {
		# noop
	    } elsif (/Sent/) {
		# noop
	    } elsif (/Recv/) {
		# noop
	    } else  {
		die "$_\n";
	    }
	}
	close(FH);
    }

    $result;
}

for my $dir (@ARGV) {
    my @subdirs = glob("$dir/*");

    die unless @subdirs > 0;

    my $proxy_host = basename($subdirs[0]);
    my @traffic_types = glob("$subdirs[0]/*");
    my $t = Text::ASCIITable->new();
    my $result_date = basename($dir);

    $t = Text::ASCIITable->new({ headingText => "$result_date / $proxy_host" });
    $t->setCols("traffic", "Hits (sum)", "Errors (sum)", "rps (AVG)");

    my $nsamples = 0;

    for my $tt (@traffic_types) {
	my $result = parse_result_dir($tt);
	$t->addRow(basename($tt),
		   sum(@{$result->{hits}}),
		   sum(@{$result->{connection_errors}}),
		   average(@{$result->{rps}}));

	$nsamples = scalar @{$result->{rps}};
    }

    $t->setOptions("headingText", "$result_date / $nsamples samples / $proxy_host");
    print $t, "\n";
}
